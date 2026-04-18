package players

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("player not found")
var ErrCreateFailed = errors.New("create player failed")

type Player struct {
	ID          int64
	TokoID      int64
	Username    string
	ExtUsername string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByUsername(ctx context.Context, tokoID int64, username string) (*Player, error) {
	const query = `
		SELECT id, toko_id, username, ext_username
		FROM players
		WHERE toko_id = $1
			AND username = $2
		LIMIT 1
	`

	var player Player
	err := r.db.QueryRow(ctx, query, tokoID, strings.ToLower(strings.TrimSpace(username))).Scan(
		&player.ID,
		&player.TokoID,
		&player.Username,
		&player.ExtUsername,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find player by username: %w", err)
	}

	return &player, nil
}

func (r *Repository) UsernameMapForToko(ctx context.Context, tokoID int64) (map[string]string, error) {
	const query = `
		SELECT ext_username, username
		FROM players
		WHERE toko_id = $1
	`

	rows, err := r.db.Query(ctx, query, tokoID)
	if err != nil {
		return nil, fmt.Errorf("query username map: %w", err)
	}
	defer rows.Close()

	usernameMap := make(map[string]string)
	for rows.Next() {
		var extUsername string
		var username string
		if err := rows.Scan(&extUsername, &username); err != nil {
			return nil, fmt.Errorf("scan username map: %w", err)
		}

		usernameMap[extUsername] = username
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate username map: %w", err)
	}

	return usernameMap, nil
}

func (r *Repository) Create(ctx context.Context, tokoID int64, username string, extUsername string) (*Player, error) {
	const query = `
		INSERT INTO players (toko_id, username, ext_username, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, toko_id, username, ext_username
	`

	var player Player
	err := r.db.QueryRow(ctx, query, tokoID, strings.ToLower(strings.TrimSpace(username)), extUsername).Scan(
		&player.ID,
		&player.TokoID,
		&player.Username,
		&player.ExtUsername,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, ErrCreateFailed
		}
		return nil, fmt.Errorf("create player: %w", err)
	}

	return &player, nil
}
