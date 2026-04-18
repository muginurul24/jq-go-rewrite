package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindUserByLogin(ctx context.Context, login string) (*User, error) {
	const query = `
		SELECT id, username, name, email, password, role, is_active, app_authentication_secret, app_authentication_recovery_codes
		FROM users
		WHERE username = $1 OR LOWER(email) = LOWER($1)
		LIMIT 1
	`

	var user User
	err := r.db.QueryRow(ctx, query, strings.TrimSpace(login)).Scan(
		&user.ID,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.MFASecret,
		&user.MFARecovery,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user by login: %w", err)
	}

	return &user, nil
}

func (r *Repository) FindUserByID(ctx context.Context, id int64) (*User, error) {
	const query = `
		SELECT id, username, name, email, password, role, is_active, app_authentication_secret, app_authentication_recovery_codes
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	var user User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.MFASecret,
		&user.MFARecovery,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return &user, nil
}

func (r *Repository) CreateUser(ctx context.Context, input RegisterInput, passwordHash string) (*User, error) {
	const query = `
		INSERT INTO users (
			username,
			name,
			email,
			password,
			role,
			is_active,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, 'user', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, username, name, email, password, role, is_active, app_authentication_secret, app_authentication_recovery_codes
	`

	var user User
	err := r.db.QueryRow(
		ctx,
		query,
		input.Username,
		input.Name,
		input.Email,
		passwordHash,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.MFASecret,
		&user.MFARecovery,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicateUser
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &user, nil
}

func (r *Repository) UpdateUserMFA(ctx context.Context, userID int64, encryptedSecret string, encryptedRecoveryCodes string) error {
	const query = `
		UPDATE users
		SET
			app_authentication_secret = $2,
			app_authentication_recovery_codes = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	if _, err := r.db.Exec(ctx, query, userID, encryptedSecret, encryptedRecoveryCodes); err != nil {
		return fmt.Errorf("update user mfa: %w", err)
	}

	return nil
}

func (r *Repository) ClearUserMFA(ctx context.Context, userID int64) error {
	const query = `
		UPDATE users
		SET
			app_authentication_secret = NULL,
			app_authentication_recovery_codes = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	if _, err := r.db.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("clear user mfa: %w", err)
	}

	return nil
}

func (r *Repository) FindAccessTokenByHash(ctx context.Context, tokenHash string) (*AccessToken, error) {
	const query = `
		SELECT
			pat.id,
			pat.tokenable_type,
			pat.tokenable_id,
			pat.name,
			pat.expires_at
		FROM personal_access_tokens pat
		WHERE pat.token = $1
		LIMIT 1
	`

	return r.scanAccessToken(ctx, query, tokenHash)
}

func (r *Repository) FindAccessTokenByIDAndHash(ctx context.Context, tokenID int64, tokenHash string) (*AccessToken, error) {
	const query = `
		SELECT
			pat.id,
			pat.tokenable_type,
			pat.tokenable_id,
			pat.name,
			pat.expires_at
		FROM personal_access_tokens pat
		WHERE pat.id = $1
			AND pat.token = $2
		LIMIT 1
	`

	return r.scanAccessToken(ctx, query, tokenID, tokenHash)
}

func (r *Repository) TouchPersonalAccessToken(ctx context.Context, tokenID int64, usedAt time.Time) error {
	const query = `
		UPDATE personal_access_tokens
		SET last_used_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	if _, err := r.db.Exec(ctx, query, tokenID, usedAt); err != nil {
		return fmt.Errorf("touch personal access token: %w", err)
	}

	return nil
}

func (r *Repository) FindTokoByID(ctx context.Context, id int64) (*Toko, error) {
	const query = `
		SELECT id, user_id, name, callback_url, token, is_active
		FROM tokos
		WHERE id = $1
		LIMIT 1
	`

	var toko Toko
	var callbackURL *string
	var token *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&toko.ID,
		&toko.UserID,
		&toko.Name,
		&callbackURL,
		&token,
		&toko.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("find toko by id: %w", err)
	}

	toko.CallbackURL = callbackURL
	toko.Token = token
	return &toko, nil
}

func (r *Repository) scanAccessToken(ctx context.Context, query string, args ...any) (*AccessToken, error) {
	var token AccessToken

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&token.ID,
		&token.TokenableType,
		&token.TokenableID,
		&token.Name,
		&token.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("find access token: %w", err)
	}

	return &token, nil
}
