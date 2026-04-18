package balances

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Balance struct {
	ID       int64
	TokoID   int64
	Settle   int64
	Pending  int64
	NexusGGR int64
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindOrCreateByTokoID(ctx context.Context, tokoID int64) (*Balance, error) {
	balance, err := r.findByTokoID(ctx, tokoID)
	if err == nil {
		return balance, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("find balance by toko id: %w", err)
	}

	const insertQuery = `
		INSERT INTO balances (toko_id, settle, pending, nexusggr, created_at, updated_at)
		VALUES ($1, 0, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, toko_id, settle::bigint, pending::bigint, nexusggr::bigint
	`

	var createdBalance Balance
	err = r.db.QueryRow(ctx, insertQuery, tokoID).Scan(
		&createdBalance.ID,
		&createdBalance.TokoID,
		&createdBalance.Settle,
		&createdBalance.Pending,
		&createdBalance.NexusGGR,
	)
	if err != nil {
		return nil, fmt.Errorf("create balance for toko: %w", err)
	}

	return &createdBalance, nil
}

func (r *Repository) findByTokoID(ctx context.Context, tokoID int64) (*Balance, error) {
	const query = `
		SELECT id, toko_id, settle::bigint, pending::bigint, nexusggr::bigint
		FROM balances
		WHERE toko_id = $1
		ORDER BY id
		LIMIT 1
	`

	var balance Balance
	err := r.db.QueryRow(ctx, query, tokoID).Scan(
		&balance.ID,
		&balance.TokoID,
		&balance.Settle,
		&balance.Pending,
		&balance.NexusGGR,
	)
	if err != nil {
		return nil, err
	}

	return &balance, nil
}
