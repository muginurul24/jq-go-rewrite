package tokos

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
)

var (
	ErrNotFound             = errors.New("toko not found")
	ErrInvalidOwner         = errors.New("invalid owner")
	ErrDuplicateCallbackURL = errors.New("duplicate callback url")
)

type CreateInput struct {
	UserID      int64
	Name        string
	CallbackURL *string
	IsActive    bool
}

type UpdateInput struct {
	UserID      int64
	Name        string
	CallbackURL *string
	IsActive    bool
}

type Service struct {
	db         *pgxpool.Pool
	repository *Repository
}

func NewService(db *pgxpool.Pool, repository *Repository) *Service {
	return &Service{
		db:         db,
		repository: repository,
	}
}

func (s *Service) ListForBackoffice(ctx context.Context, user auth.PublicUser, input AdminListInput) (*AdminListResult, error) {
	return s.repository.ListForBackoffice(ctx, user, input)
}

func (s *Service) FindDetailForBackoffice(ctx context.Context, user auth.PublicUser, tokoID int64) (*AdminTokoRecord, error) {
	return s.repository.FindDetailForBackoffice(ctx, user, tokoID)
}

func (s *Service) CreateForBackoffice(ctx context.Context, actor auth.PublicUser, input CreateInput) (*AdminTokoRecord, error) {
	ownerID, err := s.resolveOwnerID(ctx, actor, input.UserID)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin create toko tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var tokoID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO tokos (user_id, name, callback_url, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, ownerID, strings.TrimSpace(input.Name), normalizeCallbackURL(input.CallbackURL), input.IsActive).Scan(&tokoID)
	if err != nil {
		return nil, mapWriteError("create toko", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO balances (toko_id, settle, pending, nexusggr, created_at, updated_at)
		VALUES ($1, 0, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, tokoID); err != nil {
		return nil, fmt.Errorf("create toko balance: %w", err)
	}

	if _, err := s.issueAPITokenTx(ctx, tx, tokoID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create toko: %w", err)
	}

	return s.repository.FindDetailForBackoffice(ctx, actor, tokoID)
}

func (s *Service) UpdateForBackoffice(ctx context.Context, actor auth.PublicUser, tokoID int64, input UpdateInput) (*AdminTokoRecord, error) {
	existing, err := s.repository.FindDetailForBackoffice(ctx, actor, tokoID)
	if err != nil {
		return nil, err
	}

	ownerID, err := s.resolveOwnerID(ctx, actor, input.UserID)
	if err != nil {
		return nil, err
	}
	if !isGlobalRole(actor.Role) {
		ownerID = existing.UserID
	}

	_, err = s.db.Exec(ctx, `
		UPDATE tokos
		SET user_id = $2,
			name = $3,
			callback_url = $4,
			is_active = $5,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL
	`, tokoID, ownerID, strings.TrimSpace(input.Name), normalizeCallbackURL(input.CallbackURL), input.IsActive)
	if err != nil {
		return nil, mapWriteError("update toko", err)
	}

	return s.repository.FindDetailForBackoffice(ctx, actor, tokoID)
}

func (s *Service) RegenerateTokenForBackoffice(ctx context.Context, actor auth.PublicUser, tokoID int64) (*AdminTokoRecord, error) {
	if _, err := s.repository.FindDetailForBackoffice(ctx, actor, tokoID); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin regenerate token tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := s.issueAPITokenTx(ctx, tx, tokoID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit regenerate token: %w", err)
	}

	return s.repository.FindDetailForBackoffice(ctx, actor, tokoID)
}

func (s *Service) AssignableOwners(ctx context.Context, actor auth.PublicUser) ([]AdminOwnerOption, error) {
	return s.repository.AssignableOwners(ctx, actor)
}

func (s *Service) resolveOwnerID(ctx context.Context, actor auth.PublicUser, requestedOwnerID int64) (int64, error) {
	if !isGlobalRole(actor.Role) {
		return actor.ID, nil
	}

	if requestedOwnerID <= 0 {
		return 0, ErrInvalidOwner
	}

	options, err := s.repository.AssignableOwners(ctx, actor)
	if err != nil {
		return 0, err
	}

	for _, option := range options {
		if option.ID == requestedOwnerID {
			return requestedOwnerID, nil
		}
	}

	return 0, ErrInvalidOwner
}

func (s *Service) issueAPITokenTx(ctx context.Context, tx pgx.Tx, tokoID int64) (string, error) {
	if _, err := tx.Exec(ctx, `
		DELETE FROM personal_access_tokens
		WHERE tokenable_type = $1 AND tokenable_id = $2
	`, auth.TokenableTypeToko, tokoID); err != nil {
		return "", fmt.Errorf("delete existing toko tokens: %w", err)
	}

	plainToken, err := randomPlainToken()
	if err != nil {
		return "", err
	}

	var tokenID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO personal_access_tokens (
			tokenable_type,
			tokenable_id,
			name,
			token,
			abilities,
			created_at,
			updated_at
		) VALUES ($1, $2, 'api', $3, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, auth.TokenableTypeToko, tokoID, sha256Hex(plainToken)).Scan(&tokenID)
	if err != nil {
		return "", fmt.Errorf("insert personal access token: %w", err)
	}

	composedToken := fmt.Sprintf("%d|%s", tokenID, plainToken)
	if _, err := tx.Exec(ctx, `
		UPDATE tokos
		SET token = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, tokoID, composedToken); err != nil {
		return "", fmt.Errorf("update toko token: %w", err)
	}

	return composedToken, nil
}

func normalizeCallbackURL(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	normalized := "https://" + trimmed
	return &normalized
}

func mapWriteError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "callback_url") {
		return ErrDuplicateCallbackURL
	}

	return fmt.Errorf("%s: %w", action, err)
}

func randomPlainToken() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read token random bytes: %w", err)
	}

	return hex.EncodeToString(buf), nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
