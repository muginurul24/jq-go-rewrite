package banks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	qrisintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/qris"
)

var (
	ErrNotFound               = errors.New("bank not found")
	ErrDuplicateAccountNumber = errors.New("duplicate bank account number")
	ErrOwnerRequired          = errors.New("bank owner required")
	ErrInvalidBankCode        = errors.New("invalid bank code")
	ErrInquiryFailed          = errors.New("bank inquiry failed")
)

var bankCatalog = map[string]string{
	"002": "BRI",
	"008": "Mandiri",
	"009": "BNI",
	"014": "BCA",
	"501": "Blu BCA Digital",
	"022": "CIMB",
	"013": "Permata",
	"111": "DKI",
	"451": "BSI",
	"542": "JAGO",
	"490": "NEO",
}

const inquiryValidationAmount int64 = 25_000

type qrisClient interface {
	Inquiry(ctx context.Context, amount int64, bankCode string, accountNumber string, transferType int) (*qrisintegration.InquiryResponse, error)
}

type AdminListInput struct {
	Search  string
	OwnerID int64
	Page    int
	PerPage int
}

type OwnerOption struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type AdminBankRecord struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"userId"`
	OwnerUsername string    `json:"ownerUsername"`
	OwnerName     string    `json:"ownerName"`
	BankCode      string    `json:"bankCode"`
	BankName      string    `json:"bankName"`
	AccountNumber string    `json:"accountNumber"`
	AccountName   string    `json:"accountName"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type AdminListResult struct {
	Data       []AdminBankRecord
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
	Owners     []OwnerOption
}

type CreateInput struct {
	UserID        int64
	BankCode      string
	AccountNumber string
	AccountName   string
}

type UpdateInput struct {
	UserID        int64
	BankCode      string
	AccountNumber string
	AccountName   string
}

type InquiryResult struct {
	BankCode      string `json:"bankCode"`
	BankName      string `json:"bankName"`
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
}

type Service struct {
	db   *pgxpool.Pool
	qris qrisClient
}

func NewService(db *pgxpool.Pool, qrisClient qrisClient) *Service {
	return &Service{
		db:   db,
		qris: qrisClient,
	}
}

func (s *Service) ListForBackoffice(ctx context.Context, actor auth.PublicUser, input AdminListInput) (*AdminListResult, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}

	perPage := input.PerPage
	switch perPage {
	case 10, 25, 50, 100:
	default:
		perPage = 25
	}

	whereClause, args := s.buildWhereClause(actor, input)

	var total int64
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM banks b
		INNER JOIN users u ON u.id = b.user_id
	`+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count banks: %w", err)
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, perPage, (page-1)*perPage)

	rows, err := s.db.Query(ctx, `
		SELECT
			b.id,
			b.user_id,
			u.username,
			u.name,
			b.bank_code,
			b.bank_name,
			b.account_number,
			b.account_name,
			b.created_at,
			b.updated_at
		FROM banks b
		INNER JOIN users u ON u.id = b.user_id
	`+whereClause+`
		ORDER BY b.created_at DESC, b.id DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)+1)+`
		OFFSET $`+fmt.Sprintf("%d", len(args)+2), listArgs...)
	if err != nil {
		return nil, fmt.Errorf("list banks: %w", err)
	}
	defer rows.Close()

	records := make([]AdminBankRecord, 0, perPage)
	for rows.Next() {
		var item AdminBankRecord
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.OwnerUsername,
			&item.OwnerName,
			&item.BankCode,
			&item.BankName,
			&item.AccountNumber,
			&item.AccountName,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan bank row: %w", err)
		}
		records = append(records, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bank rows: %w", err)
	}

	owners, err := s.accessibleOwners(ctx, actor)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}

	return &AdminListResult{
		Data:       records,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		Owners:     owners,
	}, nil
}

func (s *Service) CreateForBackoffice(ctx context.Context, actor auth.PublicUser, input CreateInput) (*AdminBankRecord, error) {
	resolvedUserID, err := s.resolveOwnerID(ctx, actor, input.UserID)
	if err != nil {
		return nil, err
	}

	bankName, err := s.resolveBankName(input.BankCode)
	if err != nil {
		return nil, err
	}

	var bankID int64
	if err := s.db.QueryRow(ctx, `
		INSERT INTO banks (
			user_id,
			bank_code,
			bank_name,
			account_number,
			account_name,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, resolvedUserID, strings.TrimSpace(input.BankCode), bankName, strings.TrimSpace(input.AccountNumber), strings.TrimSpace(input.AccountName)).Scan(&bankID); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateAccountNumber
		}
		return nil, fmt.Errorf("create bank: %w", err)
	}

	return s.findDetail(ctx, actor, bankID)
}

func (s *Service) UpdateForBackoffice(ctx context.Context, actor auth.PublicUser, bankID int64, input UpdateInput) (*AdminBankRecord, error) {
	record, err := s.findDetail(ctx, actor, bankID)
	if err != nil {
		return nil, err
	}

	resolvedUserID := record.UserID
	if isGlobalRole(actor.Role) {
		if input.UserID <= 0 {
			return nil, ErrOwnerRequired
		}
		resolvedUserID = input.UserID
	}

	bankName, err := s.resolveBankName(input.BankCode)
	if err != nil {
		return nil, err
	}

	commandTag, err := s.db.Exec(ctx, `
		UPDATE banks
		SET
			user_id = $2,
			bank_code = $3,
			bank_name = $4,
			account_number = $5,
			account_name = $6,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL
	`, bankID, resolvedUserID, strings.TrimSpace(input.BankCode), bankName, strings.TrimSpace(input.AccountNumber), strings.TrimSpace(input.AccountName))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateAccountNumber
		}
		return nil, fmt.Errorf("update bank: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return s.findDetail(ctx, actor, bankID)
}

func (s *Service) InquiryForBackoffice(ctx context.Context, actor auth.PublicUser, userID int64, bankCode string, accountNumber string) (*InquiryResult, error) {
	if isGlobalRole(actor.Role) {
		if userID <= 0 {
			return nil, ErrOwnerRequired
		}
	} else {
		userID = actor.ID
	}

	if _, err := s.resolveBankName(bankCode); err != nil {
		return nil, err
	}

	if !isGlobalRole(actor.Role) && !s.userOwnsAnyToko(ctx, userID) {
		return nil, ErrNotFound
	}

	response, err := s.qris.Inquiry(ctx, inquiryValidationAmount, bankCode, accountNumber, 2)
	if err != nil || response == nil || !response.Status || response.Data == nil {
		return nil, ErrInquiryFailed
	}

	accountName := strings.TrimSpace(response.Data.AccountName)
	if accountName == "" {
		return nil, ErrInquiryFailed
	}

	return &InquiryResult{
		BankCode:      strings.TrimSpace(bankCode),
		BankName:      strings.TrimSpace(response.Data.BankName),
		AccountNumber: strings.TrimSpace(accountNumber),
		AccountName:   accountName,
	}, nil
}

func (s *Service) findDetail(ctx context.Context, actor auth.PublicUser, bankID int64) (*AdminBankRecord, error) {
	query := `
		SELECT
			b.id,
			b.user_id,
			u.username,
			u.name,
			b.bank_code,
			b.bank_name,
			b.account_number,
			b.account_name,
			b.created_at,
			b.updated_at
		FROM banks b
		INNER JOIN users u ON u.id = b.user_id
		WHERE b.id = $1
			AND b.deleted_at IS NULL
	`
	args := []any{bankID}

	if !isGlobalRole(actor.Role) {
		query += " AND b.user_id = $2"
		args = append(args, actor.ID)
	}

	var item AdminBankRecord
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&item.UserID,
		&item.OwnerUsername,
		&item.OwnerName,
		&item.BankCode,
		&item.BankName,
		&item.AccountNumber,
		&item.AccountName,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find bank detail: %w", err)
	}

	return &item, nil
}

func (s *Service) buildWhereClause(actor auth.PublicUser, input AdminListInput) (string, []any) {
	clauses := []string{"WHERE b.deleted_at IS NULL"}
	args := []any{}

	if search := strings.TrimSpace(input.Search); search != "" {
		clauses = append(clauses, fmt.Sprintf(
			"(u.name ILIKE $%d OR u.username ILIKE $%d OR b.bank_name ILIKE $%d OR b.account_number ILIKE $%d OR b.account_name ILIKE $%d)",
			len(args)+1,
			len(args)+1,
			len(args)+1,
			len(args)+1,
			len(args)+1,
		))
		args = append(args, "%"+search+"%")
	}

	if input.OwnerID > 0 {
		clauses = append(clauses, fmt.Sprintf("b.user_id = $%d", len(args)+1))
		args = append(args, input.OwnerID)
	}

	if !isGlobalRole(actor.Role) {
		clauses = append(clauses, fmt.Sprintf("b.user_id = $%d", len(args)+1))
		args = append(args, actor.ID)
	}

	return " " + strings.Join(clauses, " AND "), args
}

func (s *Service) accessibleOwners(ctx context.Context, actor auth.PublicUser) ([]OwnerOption, error) {
	query := `
		SELECT id, username, name
		FROM users
		WHERE role NOT IN ('dev', 'superadmin')
	`
	args := []any{}

	if !isGlobalRole(actor.Role) {
		query += " AND id = $1"
		args = append(args, actor.ID)
	}

	query += " ORDER BY username ASC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list bank owners: %w", err)
	}
	defer rows.Close()

	result := make([]OwnerOption, 0, 8)
	for rows.Next() {
		var item OwnerOption
		if err := rows.Scan(&item.ID, &item.Username, &item.Name); err != nil {
			return nil, fmt.Errorf("scan bank owner: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bank owners: %w", err)
	}

	return result, nil
}

func (s *Service) resolveOwnerID(ctx context.Context, actor auth.PublicUser, requestedUserID int64) (int64, error) {
	if !isGlobalRole(actor.Role) {
		return actor.ID, nil
	}

	if requestedUserID <= 0 {
		return 0, ErrOwnerRequired
	}

	return requestedUserID, nil
}

func (s *Service) resolveBankName(bankCode string) (string, error) {
	value, ok := bankCatalog[strings.TrimSpace(bankCode)]
	if !ok {
		return "", ErrInvalidBankCode
	}

	return value, nil
}

func (s *Service) userOwnsAnyToko(ctx context.Context, userID int64) bool {
	var exists bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM tokos
			WHERE user_id = $1
				AND deleted_at IS NULL
		)
	`, userID).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func isGlobalRole(role string) bool {
	return role == "dev" || role == "superadmin"
}
