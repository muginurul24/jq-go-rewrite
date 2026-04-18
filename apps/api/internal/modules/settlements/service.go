package settlements

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Result struct {
	ProcessedCount int
	SettledTotal   int64
}

type Service struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewService(db *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger.With().Str("module", "settlements").Logger(),
	}
}

func (s *Service) SettlePendingBalances(ctx context.Context) (*Result, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id
		FROM balances
		WHERE pending > 0
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("list pending balances: %w", err)
	}
	defer rows.Close()

	var balanceIDs []int64
	for rows.Next() {
		var balanceID int64
		if err := rows.Scan(&balanceID); err != nil {
			return nil, fmt.Errorf("scan balance id: %w", err)
		}
		balanceIDs = append(balanceIDs, balanceID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate balance ids: %w", err)
	}

	result := &Result{}
	for _, balanceID := range balanceIDs {
		settledAmount, err := s.settleOne(ctx, balanceID)
		if err != nil {
			s.logger.Error().
				Err(err).
				Int64("balance_id", balanceID).
				Msg("failed to settle pending balance")
			continue
		}
		if settledAmount <= 0 {
			continue
		}

		result.ProcessedCount++
		result.SettledTotal += settledAmount
	}

	s.logger.Info().
		Int("processed_count", result.ProcessedCount).
		Int64("settled_total", result.SettledTotal).
		Msg("settled pending balances")

	return result, nil
}

func (s *Service) settleOne(ctx context.Context, balanceID int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin settlement transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var pending int64
	err = tx.QueryRow(ctx, `
		SELECT pending::bigint
		FROM balances
		WHERE id = $1
		FOR UPDATE
	`, balanceID).Scan(&pending)
	if err != nil {
		return 0, fmt.Errorf("lock balance row: %w", err)
	}

	amountToSettle := calculateSettlementAmount(pending)
	if amountToSettle <= 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit empty settlement transaction: %w", err)
		}
		committed = true
		return 0, nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE balances
		SET pending = pending - $2,
			settle = settle + $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, balanceID, amountToSettle); err != nil {
		return 0, fmt.Errorf("update settled balance: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit settlement transaction: %w", err)
	}

	committed = true
	return amountToSettle, nil
}

func calculateSettlementAmount(pending int64) int64 {
	return int64(math.Round(float64(pending) * 0.7))
}
