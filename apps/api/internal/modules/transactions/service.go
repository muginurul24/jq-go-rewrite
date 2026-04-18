package transactions

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const PendingExpiryMinutes = 30

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) ExpirePendingTransaction(ctx context.Context, transactionID int64) error {
	if _, err := s.db.Exec(ctx, `
		UPDATE transactions
		SET status = 'expired', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND status = 'pending'
			AND created_at <= CURRENT_TIMESTAMP - INTERVAL '30 minutes'
	`, transactionID); err != nil {
		return fmt.Errorf("expire pending transaction: %w", err)
	}

	return nil
}

func (s *Service) ExpireOverduePendingQRIS(ctx context.Context) (int64, error) {
	var expiredCount int64
	if err := s.db.QueryRow(ctx, `
		WITH expired AS (
			UPDATE transactions
			SET status = 'expired', updated_at = CURRENT_TIMESTAMP
			WHERE category = 'qris'
				AND type = 'deposit'
				AND status = 'pending'
				AND deleted_at IS NULL
				AND created_at <= CURRENT_TIMESTAMP - INTERVAL '30 minutes'
			RETURNING id
		)
		SELECT COUNT(*)::bigint
		FROM expired
	`).Scan(&expiredCount); err != nil {
		return 0, fmt.Errorf("expire overdue pending qris transactions: %w", err)
	}

	return expiredCount, nil
}
