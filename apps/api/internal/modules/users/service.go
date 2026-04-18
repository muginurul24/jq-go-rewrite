package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
)

var (
	ErrForbidden         = errors.New("forbidden")
	ErrNotFound          = errors.New("user not found")
	ErrDuplicateUsername = errors.New("duplicate username")
	ErrDuplicateEmail    = errors.New("duplicate email")
	ErrInvalidRole       = errors.New("invalid role")
)

type AdminListInput struct {
	Search  string
	Role    string
	Status  string
	Page    int
	PerPage int
}

type AdminUserRecord struct {
	ID        int64
	Username  string
	Name      string
	Email     string
	Role      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AdminUserSummary struct {
	TotalUsers  int64
	ActiveUsers int64
	AdminUsers  int64
	EndUsers    int64
}

type AdminListResult struct {
	Data       []AdminUserRecord
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
	Summary    AdminUserSummary
}

type CreateInput struct {
	Username string
	Name     string
	Email    string
	Role     string
	IsActive bool
	Password string
}

type UpdateInput struct {
	Username string
	Name     string
	Email    string
	Role     string
	IsActive bool
	Password *string
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) ListForBackoffice(ctx context.Context, actor auth.PublicUser, input AdminListInput) (*AdminListResult, error) {
	if !canManageUsers(actor.Role) {
		return nil, ErrForbidden
	}

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

	whereClause, args := buildWhereClause(input)

	countQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE role NOT IN ('dev', 'superadmin')
	` + whereClause

	var total int64
	if err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, perPage, (page-1)*perPage)

	listQuery := `
		SELECT id, username, name, email, role, is_active, created_at, updated_at
		FROM users
		WHERE role NOT IN ('dev', 'superadmin')
	` + whereClause + `
		ORDER BY created_at DESC, id DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + `
		OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	rows, err := s.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	items := make([]AdminUserRecord, 0, perPage)
	for rows.Next() {
		var item AdminUserRecord
		if err := rows.Scan(
			&item.ID,
			&item.Username,
			&item.Name,
			&item.Email,
			&item.Role,
			&item.IsActive,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	summary, err := s.summary(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}

	return &AdminListResult{
		Data:       items,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		Summary:    summary,
	}, nil
}

func (s *Service) FindDetailForBackoffice(ctx context.Context, actor auth.PublicUser, userID int64) (*AdminUserRecord, error) {
	if !canManageUsers(actor.Role) {
		return nil, ErrForbidden
	}

	var item AdminUserRecord
	err := s.db.QueryRow(ctx, `
		SELECT id, username, name, email, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`, userID).Scan(
		&item.ID,
		&item.Username,
		&item.Name,
		&item.Email,
		&item.Role,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user detail: %w", err)
	}

	return &item, nil
}

func (s *Service) CreateForBackoffice(ctx context.Context, actor auth.PublicUser, input CreateInput) (*AdminUserRecord, error) {
	if !canManageUsers(actor.Role) {
		return nil, ErrForbidden
	}
	if !isAllowedManagedRole(actor.Role, input.Role) {
		return nil, ErrInvalidRole
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var created AdminUserRecord
	err = s.db.QueryRow(ctx, `
		INSERT INTO users (
			username,
			name,
			email,
			password,
			role,
			is_active,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, username, name, email, role, is_active, created_at, updated_at
	`,
		strings.TrimSpace(input.Username),
		strings.TrimSpace(input.Name),
		strings.TrimSpace(input.Email),
		string(passwordHash),
		input.Role,
		input.IsActive,
	).Scan(
		&created.ID,
		&created.Username,
		&created.Name,
		&created.Email,
		&created.Role,
		&created.IsActive,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return nil, mapWriteError("create user", err)
	}

	return &created, nil
}

func (s *Service) UpdateForBackoffice(ctx context.Context, actor auth.PublicUser, userID int64, input UpdateInput) (*AdminUserRecord, error) {
	if !canManageUsers(actor.Role) {
		return nil, ErrForbidden
	}
	if !isAllowedManagedRole(actor.Role, input.Role) {
		return nil, ErrInvalidRole
	}

	existing, err := s.FindDetailForBackoffice(ctx, actor, userID)
	if err != nil {
		return nil, err
	}
	if !isAllowedManagedRole(actor.Role, existing.Role) {
		return nil, ErrForbidden
	}

	if input.Password != nil && strings.TrimSpace(*input.Password) != "" {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(*input.Password)), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash update password: %w", err)
		}

		_, err = s.db.Exec(ctx, `
			UPDATE users
			SET username = $2,
				name = $3,
				email = $4,
				role = $5,
				is_active = $6,
				password = $7,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, userID, strings.TrimSpace(input.Username), strings.TrimSpace(input.Name), strings.TrimSpace(input.Email), input.Role, input.IsActive, string(passwordHash))
		if err != nil {
			return nil, mapWriteError("update user", err)
		}
	} else {
		_, err = s.db.Exec(ctx, `
			UPDATE users
			SET username = $2,
				name = $3,
				email = $4,
				role = $5,
				is_active = $6,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, userID, strings.TrimSpace(input.Username), strings.TrimSpace(input.Name), strings.TrimSpace(input.Email), input.Role, input.IsActive)
		if err != nil {
			return nil, mapWriteError("update user", err)
		}
	}

	return s.FindDetailForBackoffice(ctx, actor, userID)
}

func (s *Service) summary(ctx context.Context, whereClause string, args []any) (AdminUserSummary, error) {
	query := `
		SELECT
			COUNT(*) AS total_users,
			COUNT(*) FILTER (WHERE is_active) AS active_users,
			COUNT(*) FILTER (WHERE role = 'admin') AS admin_users,
			COUNT(*) FILTER (WHERE role = 'user') AS end_users
		FROM users
		WHERE role NOT IN ('dev', 'superadmin')
	` + whereClause

	var summary AdminUserSummary
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&summary.TotalUsers,
		&summary.ActiveUsers,
		&summary.AdminUsers,
		&summary.EndUsers,
	); err != nil {
		return AdminUserSummary{}, fmt.Errorf("summary users: %w", err)
	}

	return summary, nil
}

func buildWhereClause(input AdminListInput) (string, []any) {
	parts := make([]string, 0, 3)
	args := make([]any, 0, 3)

	if search := strings.TrimSpace(input.Search); search != "" {
		args = append(args, "%"+search+"%")
		index := len(args)
		parts = append(parts, fmt.Sprintf(
			"AND (username ILIKE $%[1]d OR name ILIKE $%[1]d OR email ILIKE $%[1]d OR role ILIKE $%[1]d)",
			index,
		))
	}

	switch strings.ToLower(strings.TrimSpace(input.Role)) {
	case "admin", "user":
		args = append(args, input.Role)
		parts = append(parts, fmt.Sprintf("AND role = $%d", len(args)))
	}

	switch strings.ToLower(strings.TrimSpace(input.Status)) {
	case "active":
		parts = append(parts, "AND is_active = TRUE")
	case "inactive":
		parts = append(parts, "AND is_active = FALSE")
	}

	if len(parts) == 0 {
		return "", args
	}

	return "\n" + strings.Join(parts, "\n"), args
}

func mapWriteError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if strings.Contains(pgErr.ConstraintName, "username") {
			return ErrDuplicateUsername
		}
		if strings.Contains(pgErr.ConstraintName, "email") {
			return ErrDuplicateEmail
		}
	}

	return fmt.Errorf("%s: %w", action, err)
}

func canManageUsers(role string) bool {
	return role == "dev" || role == "superadmin"
}

func isAllowedManagedRole(actorRole string, targetRole string) bool {
	switch actorRole {
	case "dev":
		return targetRole == "dev" || targetRole == "superadmin" || targetRole == "admin" || targetRole == "user"
	case "superadmin":
		return targetRole == "admin" || targetRole == "user"
	default:
		return false
	}
}
