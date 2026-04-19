package operationalfees

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const MonthlyOperationalFeeAmount int64 = 100_000

type Result struct {
	ProcessedCount int
	DeductedTotal  int64
}

type Service struct {
	db            *pgxpool.Pool
	logger        zerolog.Logger
	notifications notificationWriter
}

type notificationWriter interface {
	NotifyMonthlyOperationalFeesCollected(ctx context.Context, processedCount int, deductedTotal int64) error
}

type incomeRecord struct {
	ID int64
}

type balanceRecord struct {
	ID     int64
	Settle int64
}

func NewService(db *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger.With().Str("module", "operational_fees").Logger(),
	}
}

func (s *Service) WithNotifications(service notificationWriter) *Service {
	s.notifications = service
	return s
}

func (s *Service) CollectMonthlyOperationalFees(ctx context.Context) (*Result, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin operational fee transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	income, err := lockIncome(ctx, tx)
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT id, COALESCE(settle, 0)::bigint
		FROM balances
		WHERE COALESCE(settle, 0) > 0
		ORDER BY id
		FOR UPDATE
	`)
	if err != nil {
		return nil, fmt.Errorf("list balances for operational fees: %w", err)
	}

	balances := make([]balanceRecord, 0)
	result := &Result{}
	for rows.Next() {
		var balance balanceRecord
		if err := rows.Scan(&balance.ID, &balance.Settle); err != nil {
			return nil, fmt.Errorf("scan balance for operational fees: %w", err)
		}
		balances = append(balances, balance)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate balances for operational fees: %w", err)
	}
	rows.Close()

	for _, balance := range balances {
		deduction := calculateOperationalFeeDeduction(balance.Settle)
		if deduction <= 0 {
			continue
		}

		if _, err := tx.Exec(ctx, `
			UPDATE balances
			SET settle = settle - $2,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, balance.ID, deduction); err != nil {
			return nil, fmt.Errorf("deduct operational fee from settle: %w", err)
		}

		result.ProcessedCount++
		result.DeductedTotal += deduction
	}

	if result.DeductedTotal > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE incomes
			SET amount = amount + $2,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, income.ID, result.DeductedTotal); err != nil {
			return nil, fmt.Errorf("increment income by operational fees: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit operational fee transaction: %w", err)
	}
	committed = true

	s.logger.Info().
		Int("processed_count", result.ProcessedCount).
		Int64("deducted_total", result.DeductedTotal).
		Msg("collected monthly operational fees")

	if s.notifications != nil && result.DeductedTotal > 0 {
		_ = s.notifications.NotifyMonthlyOperationalFeesCollected(ctx, result.ProcessedCount, result.DeductedTotal)
	}

	return result, nil
}

func calculateOperationalFeeDeduction(settle int64) int64 {
	switch {
	case settle <= 0:
		return 0
	case settle < MonthlyOperationalFeeAmount:
		return settle
	default:
		return MonthlyOperationalFeeAmount
	}
}

func lockIncome(ctx context.Context, tx pgx.Tx) (*incomeRecord, error) {
	const selectQuery = `
		SELECT id
		FROM incomes
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`

	var income incomeRecord
	err := tx.QueryRow(ctx, selectQuery).Scan(&income.ID)
	if err == nil {
		return &income, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lock income: %w", err)
	}

	const insertQuery = `
		INSERT INTO incomes (ggr, fee_transaction, fee_withdrawal, amount, created_at, updated_at)
		VALUES (7, 3, 15, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	if err := tx.QueryRow(ctx, insertQuery).Scan(&income.ID); err != nil {
		return nil, fmt.Errorf("create income row: %w", err)
	}

	return &income, nil
}
