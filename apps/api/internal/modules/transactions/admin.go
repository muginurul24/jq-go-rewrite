package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
)

type AdminListInput struct {
	Search        string
	Categories    []string
	Types         []string
	Statuses      []string
	TokoIDs       []int64
	DateFrom      *string
	DateUntil     *string
	AmountMin     *int64
	AmountMax     *int64
	Page          int
	PerPage       int
	SortBy        string
	SortDirection string
}

type AdminListResult struct {
	Items       []AdminTransactionRecord
	Page        int
	PerPage     int
	Total       int64
	TotalPages  int64
	TotalAmount int64
	TokoOptions []AdminTokoOption
}

type AdminTransactionRecord struct {
	ID             int64
	TokoID         int64
	TokoName       string
	OwnerUsername  string
	Player         *string
	ExternalPlayer *string
	Category       string
	Type           string
	Status         string
	Amount         int64
	Code           *string
	Note           *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AdminTransactionDetail struct {
	Record      AdminTransactionRecord
	NoteSummary *string
	NotePayload string
}

type AdminTokoOption struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	OwnerUsername string `json:"ownerUsername"`
}

func (s *Service) ListForBackoffice(ctx context.Context, user auth.PublicUser, input AdminListInput) (*AdminListResult, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}

	perPage := input.PerPage
	switch perPage {
	case 25, 50, 100:
	default:
		perPage = 25
	}

	baseQuery, args := buildAdminTransactionBaseQuery(user, input)
	orderBy := buildAdminTransactionOrderBy(input.SortBy, input.SortDirection)

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int64
	if err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count transactions for backoffice: %w", err)
	}

	sumQuery := "SELECT COALESCE(SUM(tx.amount), 0) " + baseQuery
	var totalAmount int64
	if err := s.db.QueryRow(ctx, sumQuery, args...).Scan(&totalAmount); err != nil {
		return nil, fmt.Errorf("sum transactions for backoffice: %w", err)
	}

	records, err := s.queryAdminTransactionRecords(ctx, baseQuery, args, orderBy, &perPage, intPtr((page-1)*perPage))
	if err != nil {
		return nil, err
	}

	tokoOptions, err := s.listScopedTokoOptions(ctx, user)
	if err != nil {
		return nil, err
	}

	return &AdminListResult{
		Items:       records,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  int64(math.Ceil(float64(total) / float64(perPage))),
		TotalAmount: totalAmount,
		TokoOptions: tokoOptions,
	}, nil
}

func (s *Service) ExportForBackoffice(ctx context.Context, user auth.PublicUser, input AdminListInput) ([]AdminTransactionRecord, error) {
	baseQuery, args := buildAdminTransactionBaseQuery(user, input)
	orderBy := buildAdminTransactionOrderBy(input.SortBy, input.SortDirection)

	records, err := s.queryAdminTransactionRecords(ctx, baseQuery, args, orderBy, nil, nil)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (s *Service) FindDetailForBackoffice(ctx context.Context, user auth.PublicUser, transactionID int64) (*AdminTransactionDetail, error) {
	baseQuery, args := buildAdminTransactionBaseQuery(user, AdminListInput{})
	args = append(args, transactionID)

	query := `
		SELECT
			tx.id,
			tx.toko_id,
			t.name,
			u.username,
			tx.player,
			tx.external_player,
			tx.category,
			tx.type,
			tx.status,
			tx.amount,
			tx.code,
			tx.note,
			tx.created_at,
			tx.updated_at
	` + baseQuery + `
			AND tx.id = $` + fmt.Sprint(len(args)) + `
		LIMIT 1`

	var record AdminTransactionRecord
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&record.ID,
		&record.TokoID,
		&record.TokoName,
		&record.OwnerUsername,
		&record.Player,
		&record.ExternalPlayer,
		&record.Category,
		&record.Type,
		&record.Status,
		&record.Amount,
		&record.Code,
		&record.Note,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("find transaction detail for backoffice: %w", err)
	}

	return &AdminTransactionDetail{
		Record:      record,
		NoteSummary: summarizeAdminNote(record.Note),
		NotePayload: notePayload(record.Note),
	}, nil
}

func (s *Service) queryAdminTransactionRecords(
	ctx context.Context,
	baseQuery string,
	args []any,
	orderBy string,
	limit *int,
	offset *int,
) ([]AdminTransactionRecord, error) {
	queryArgs := append([]any{}, args...)
	query := `
		SELECT
			tx.id,
			tx.toko_id,
			t.name,
			u.username,
			tx.player,
			tx.external_player,
			tx.category,
			tx.type,
			tx.status,
			tx.amount,
			tx.code,
			tx.note,
			tx.created_at,
			tx.updated_at
	` + baseQuery + `
	` + orderBy

	if limit != nil {
		queryArgs = append(queryArgs, *limit)
		query += `
		LIMIT $` + fmt.Sprint(len(queryArgs))
	}

	if offset != nil {
		queryArgs = append(queryArgs, *offset)
		query += `
		OFFSET $` + fmt.Sprint(len(queryArgs))
	}

	rows, err := s.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list transactions for backoffice: %w", err)
	}
	defer rows.Close()

	capacity := 0
	if limit != nil && *limit > 0 {
		capacity = *limit
	}
	records := make([]AdminTransactionRecord, 0, capacity)
	for rows.Next() {
		var record AdminTransactionRecord
		if err := rows.Scan(
			&record.ID,
			&record.TokoID,
			&record.TokoName,
			&record.OwnerUsername,
			&record.Player,
			&record.ExternalPlayer,
			&record.Category,
			&record.Type,
			&record.Status,
			&record.Amount,
			&record.Code,
			&record.Note,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan transaction record: %w", err)
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transaction records: %w", err)
	}

	return records, nil
}

func (s *Service) listScopedTokoOptions(ctx context.Context, user auth.PublicUser) ([]AdminTokoOption, error) {
	args := []any{}
	query := `
		SELECT t.id, t.name, u.username
		FROM tokos t
		INNER JOIN users u ON u.id = t.user_id
		WHERE t.deleted_at IS NULL
	`

	if !isBackofficeGlobalRole(user.Role) {
		args = append(args, user.ID)
		query += fmt.Sprintf(" AND t.user_id = $%d", len(args))
	}

	query += " ORDER BY t.name ASC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list scoped toko options: %w", err)
	}
	defer rows.Close()

	options := make([]AdminTokoOption, 0)
	for rows.Next() {
		var option AdminTokoOption
		if err := rows.Scan(&option.ID, &option.Name, &option.OwnerUsername); err != nil {
			return nil, fmt.Errorf("scan scoped toko option: %w", err)
		}

		options = append(options, option)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scoped toko options: %w", err)
	}

	return options, nil
}

func buildAdminTransactionBaseQuery(user auth.PublicUser, input AdminListInput) (string, []any) {
	args := []any{}
	where := []string{"tx.deleted_at IS NULL"}

	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if !isBackofficeGlobalRole(user.Role) {
		where = append(where, "t.user_id = "+addArg(user.ID))
	}

	search := strings.TrimSpace(input.Search)
	if search != "" {
		placeholder := addArg("%" + search + "%")
		where = append(where, fmt.Sprintf(`(
			t.name ILIKE %[1]s
			OR u.username ILIKE %[1]s
			OR COALESCE(tx.player, '') ILIKE %[1]s
			OR COALESCE(tx.external_player, '') ILIKE %[1]s
			OR COALESCE(tx.code, '') ILIKE %[1]s
		)`, placeholder))
	}

	if len(input.Categories) > 0 {
		where = append(where, "tx.category = ANY("+addArg(input.Categories)+")")
	}

	if len(input.Types) > 0 {
		where = append(where, "tx.type = ANY("+addArg(input.Types)+")")
	}

	if len(input.Statuses) > 0 {
		where = append(where, "tx.status = ANY("+addArg(input.Statuses)+")")
	}

	if len(input.TokoIDs) > 0 {
		where = append(where, "tx.toko_id = ANY("+addArg(input.TokoIDs)+")")
	}

	if input.DateFrom != nil && strings.TrimSpace(*input.DateFrom) != "" {
		where = append(where, "DATE(tx.created_at) >= "+addArg(strings.TrimSpace(*input.DateFrom)))
	}

	if input.DateUntil != nil && strings.TrimSpace(*input.DateUntil) != "" {
		where = append(where, "DATE(tx.created_at) <= "+addArg(strings.TrimSpace(*input.DateUntil)))
	}

	if input.AmountMin != nil {
		where = append(where, "tx.amount >= "+addArg(*input.AmountMin))
	}

	if input.AmountMax != nil {
		where = append(where, "tx.amount <= "+addArg(*input.AmountMax))
	}

	query := `
		FROM transactions tx
		INNER JOIN tokos t ON t.id = tx.toko_id
		INNER JOIN users u ON u.id = t.user_id
		WHERE ` + strings.Join(where, " AND ")

	return query, args
}

func buildAdminTransactionOrderBy(sortBy string, sortDirection string) string {
	sortColumn := map[string]string{
		"created_at": "tx.created_at",
		"amount":     "tx.amount",
		"status":     "tx.status",
		"toko":       "t.name",
	}[strings.TrimSpace(sortBy)]
	if sortColumn == "" {
		sortColumn = "tx.created_at"
	}

	direction := strings.ToUpper(strings.TrimSpace(sortDirection))
	if direction != "ASC" {
		direction = "DESC"
	}

	return "ORDER BY " + sortColumn + " " + direction + ", tx.id DESC"
}

func isBackofficeGlobalRole(role string) bool {
	return role == "dev" || role == "superadmin"
}

func summarizeAdminNote(note *string) *string {
	if note == nil || strings.TrimSpace(*note) == "" {
		return nil
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(*note), &decoded); err != nil {
		trimmed := strings.TrimSpace(*note)
		return &trimmed
	}

	parts := make([]string, 0, len(decoded))
	for key, value := range decoded {
		switch typed := value.(type) {
		case string:
			parts = append(parts, fmt.Sprintf("%s: %s", headline(key), typed))
		case float64:
			parts = append(parts, fmt.Sprintf("%s: %.0f", headline(key), typed))
		case bool:
			parts = append(parts, fmt.Sprintf("%s: %t", headline(key), typed))
		case nil:
			parts = append(parts, fmt.Sprintf("%s: -", headline(key)))
		}
	}

	if len(parts) == 0 {
		pretty := notePayload(note)
		return &pretty
	}

	summary := strings.Join(parts, " | ")
	return &summary
}

func notePayload(note *string) string {
	if note == nil || strings.TrimSpace(*note) == "" {
		return "{\n  \"note\": \"-\"\n}"
	}

	var decoded any
	if err := json.Unmarshal([]byte(*note), &decoded); err != nil {
		return strings.TrimSpace(*note)
	}

	pretty, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return strings.TrimSpace(*note)
	}

	return string(pretty)
}

func headline(value string) string {
	replaced := strings.ReplaceAll(value, "_", " ")
	parts := strings.Fields(replaced)
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}

	return strings.Join(parts, " ")
}

func intPtr(value int) *int {
	return &value
}
