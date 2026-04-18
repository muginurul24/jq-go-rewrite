package tokos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
)

type AdminListInput struct {
	Search  string
	Status  string
	OwnerID int64
	Page    int
	PerPage int
}

type AdminOwnerOption struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type AdminTokoRecord struct {
	ID            int64
	UserID        int64
	OwnerUsername string
	OwnerName     string
	Name          string
	CallbackURL   *string
	Token         *string
	IsActive      bool
	Pending       int64
	Settle        int64
	Nexusggr      int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AdminTokoSummary struct {
	TotalTokos    int64
	ActiveTokos   int64
	TotalPending  int64
	TotalSettle   int64
	TotalNexusggr int64
}

type AdminListResult struct {
	Data       []AdminTokoRecord
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
	Summary    AdminTokoSummary
	Owners     []AdminOwnerOption
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListForBackoffice(ctx context.Context, user auth.PublicUser, input AdminListInput) (*AdminListResult, error) {
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

	whereClause, args := buildBackofficeWhereClause(user, input)

	countQuery := `SELECT COUNT(*) ` + adminBaseFromQuery + " " + whereClause

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count tokos: %w", err)
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, perPage, (page-1)*perPage)

	listQuery := `
		SELECT
			t.id,
			t.user_id,
			u.username,
			u.name,
			t.name,
			t.callback_url,
			t.token,
			t.is_active,
			COALESCE(b.pending, 0) AS pending,
			COALESCE(b.settle, 0) AS settle,
			COALESCE(b.nexusggr, 0) AS nexusggr,
			t.created_at,
			t.updated_at
	` + adminBaseFromQuery + " " + whereClause + `
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + `
		OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	rows, err := r.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("list tokos: %w", err)
	}
	defer rows.Close()

	records := make([]AdminTokoRecord, 0, perPage)
	for rows.Next() {
		record, err := scanAdminToko(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tokos: %w", err)
	}

	summary, err := r.summaryForBackoffice(ctx, user, input)
	if err != nil {
		return nil, err
	}

	owners, err := r.accessibleOwners(ctx, user)
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
		Summary:    summary,
		Owners:     owners,
	}, nil
}

func (r *Repository) FindDetailForBackoffice(ctx context.Context, user auth.PublicUser, tokoID int64) (*AdminTokoRecord, error) {
	whereClause, args := buildBackofficeWhereClause(user, AdminListInput{})
	args = append(args, tokoID)

	query := `
		SELECT
			t.id,
			t.user_id,
			u.username,
			u.name,
			t.name,
			t.callback_url,
			t.token,
			t.is_active,
			COALESCE(b.pending, 0) AS pending,
			COALESCE(b.settle, 0) AS settle,
			COALESCE(b.nexusggr, 0) AS nexusggr,
			t.created_at,
			t.updated_at
	` + adminBaseFromQuery + " " + whereClause + `
			AND t.id = $` + fmt.Sprintf("%d", len(args))

	row := r.db.QueryRow(ctx, query, args...)
	record, err := scanAdminToko(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &record, nil
}

func (r *Repository) AssignableOwners(ctx context.Context, currentUser auth.PublicUser) ([]AdminOwnerOption, error) {
	return r.accessibleOwners(ctx, currentUser)
}

func (r *Repository) summaryForBackoffice(ctx context.Context, user auth.PublicUser, input AdminListInput) (AdminTokoSummary, error) {
	whereClause, args := buildBackofficeWhereClause(user, input)

	query := `
		SELECT
			COUNT(*) AS total_tokos,
			COUNT(*) FILTER (WHERE t.is_active) AS active_tokos,
			COALESCE(SUM(COALESCE(b.pending, 0)), 0) AS total_pending,
			COALESCE(SUM(COALESCE(b.settle, 0)), 0) AS total_settle,
			COALESCE(SUM(COALESCE(b.nexusggr, 0)), 0) AS total_nexusggr
	` + adminBaseFromQuery + " " + whereClause

	var summary AdminTokoSummary
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&summary.TotalTokos,
		&summary.ActiveTokos,
		&summary.TotalPending,
		&summary.TotalSettle,
		&summary.TotalNexusggr,
	); err != nil {
		return AdminTokoSummary{}, fmt.Errorf("summary tokos: %w", err)
	}

	return summary, nil
}

func (r *Repository) accessibleOwners(ctx context.Context, user auth.PublicUser) ([]AdminOwnerOption, error) {
	args := []any{}
	query := `
		SELECT id, username, name
		FROM users
		WHERE role NOT IN ('dev', 'superadmin')
	`

	if !isGlobalRole(user.Role) {
		query += " AND id = $1"
		args = append(args, user.ID)
	}

	query += " ORDER BY username ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list owners: %w", err)
	}
	defer rows.Close()

	options := make([]AdminOwnerOption, 0, 8)
	for rows.Next() {
		var option AdminOwnerOption
		if err := rows.Scan(&option.ID, &option.Username, &option.Name); err != nil {
			return nil, fmt.Errorf("scan owner option: %w", err)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owner options: %w", err)
	}

	return options, nil
}

const adminBaseFromQuery = `
		FROM tokos t
		INNER JOIN users u ON u.id = t.user_id
		LEFT JOIN balances b ON b.toko_id = t.id
		WHERE t.deleted_at IS NULL
	`

func buildBackofficeWhereClause(user auth.PublicUser, input AdminListInput) (string, []any) {
	args := make([]any, 0, 4)
	parts := make([]string, 0, 4)

	search := strings.TrimSpace(input.Search)
	if search != "" {
		args = append(args, "%"+search+"%")
		index := len(args)
		parts = append(parts, fmt.Sprintf(
			"(u.username ILIKE $%[1]d OR u.name ILIKE $%[1]d OR t.name ILIKE $%[1]d OR COALESCE(t.callback_url, '') ILIKE $%[1]d OR COALESCE(t.token, '') ILIKE $%[1]d)",
			index,
		))
	}

	switch strings.ToLower(strings.TrimSpace(input.Status)) {
	case "active":
		parts = append(parts, "t.is_active = TRUE")
	case "inactive":
		parts = append(parts, "t.is_active = FALSE")
	}

	if input.OwnerID > 0 {
		args = append(args, input.OwnerID)
		parts = append(parts, fmt.Sprintf("t.user_id = $%d", len(args)))
	}

	if !isGlobalRole(user.Role) {
		args = append(args, user.ID)
		parts = append(parts, fmt.Sprintf("t.user_id = $%d", len(args)))
	}

	if len(parts) == 0 {
		return "", args
	}

	return " AND " + strings.Join(parts, " AND "), args
}

func scanAdminToko(scanner interface {
	Scan(dest ...any) error
}) (AdminTokoRecord, error) {
	var record AdminTokoRecord
	if err := scanner.Scan(
		&record.ID,
		&record.UserID,
		&record.OwnerUsername,
		&record.OwnerName,
		&record.Name,
		&record.CallbackURL,
		&record.Token,
		&record.IsActive,
		&record.Pending,
		&record.Settle,
		&record.Nexusggr,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return AdminTokoRecord{}, err
	}

	return record, nil
}

func isGlobalRole(role string) bool {
	return role == "dev" || role == "superadmin"
}
