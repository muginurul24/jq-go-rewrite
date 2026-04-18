package qris

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	qrisintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/qris"
)

const defaultExpireSeconds = 300

var (
	ErrGenerateUpstreamFailure    = errors.New("generate qris upstream failure")
	ErrTransactionNotFound        = errors.New("transaction not found")
	ErrCheckStatusUpstreamFailure = errors.New("check status upstream failure")
)

type qrisClient interface {
	Generate(ctx context.Context, username string, amount int64, expire int, customRef *string) (*qrisintegration.GenerateResponse, error)
	CheckStatus(ctx context.Context, trxID string) (*qrisintegration.CheckStatusResponse, error)
}

type pendingExpiryScheduler interface {
	EnqueueExpirePendingTransaction(ctx context.Context, transactionID int64) error
}

type Service struct {
	db              *pgxpool.Pool
	client          qrisClient
	expiryScheduler pendingExpiryScheduler
	logger          zerolog.Logger
}

type GenerateParams struct {
	Username  string
	Amount    int64
	Expire    *int
	CustomRef *string
}

type GenerateResult struct {
	Data  any
	TrxID string
}

type CheckStatusResult struct {
	TrxID  string
	Status string
}

func NewService(db *pgxpool.Pool, client qrisClient, expiryScheduler pendingExpiryScheduler, logger zerolog.Logger) *Service {
	return &Service{
		db:              db,
		client:          client,
		expiryScheduler: expiryScheduler,
		logger:          logger.With().Str("module", "qris").Logger(),
	}
}

func (s *Service) Generate(ctx context.Context, toko auth.Toko, params GenerateParams) (*GenerateResult, error) {
	expire := defaultExpireSeconds
	if params.Expire != nil {
		expire = *params.Expire
	}

	upstreamResponse, err := s.client.Generate(ctx, params.Username, params.Amount, expire, params.CustomRef)
	if err != nil || !upstreamResponse.Status {
		return nil, ErrGenerateUpstreamFailure
	}

	trxID, _ := upstreamResponse.TrxID.(string)
	noteJSON, err := json.Marshal(map[string]any{
		"purpose":    "generate",
		"custom_ref": params.CustomRefValue(),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal qris note: %w", err)
	}

	var transactionID int64
	if err := s.db.QueryRow(ctx, `
		INSERT INTO transactions (
			toko_id,
			player,
			category,
			amount,
			type,
			status,
			code,
			note,
			created_at,
			updated_at
		) VALUES ($1, $2, 'qris', $3, 'deposit', 'pending', $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, toko.ID, params.Username, params.Amount, trxID, string(noteJSON)).Scan(&transactionID); err != nil {
		return nil, fmt.Errorf("insert qris transaction: %w", err)
	}

	if s.expiryScheduler != nil {
		if err := s.expiryScheduler.EnqueueExpirePendingTransaction(ctx, transactionID); err != nil {
			s.logger.Error().
				Err(err).
				Int64("transaction_id", transactionID).
				Str("trx_id", trxID).
				Msg("failed to enqueue pending transaction expiry")
		}
	}

	return &GenerateResult{
		Data:  upstreamResponse.Data,
		TrxID: trxID,
	}, nil
}

func (s *Service) CheckStatus(ctx context.Context, toko auth.Toko, trxID string) (*CheckStatusResult, error) {
	transactionID, err := s.findTransactionID(ctx, toko.ID, trxID)
	if err != nil {
		return nil, err
	}

	upstreamResponse, err := s.client.CheckStatus(ctx, trxID)
	if err != nil {
		return nil, ErrCheckStatusUpstreamFailure
	}

	status := normalizeTransactionStatus(upstreamResponse.Status)
	if status == "" {
		return nil, ErrCheckStatusUpstreamFailure
	}

	if _, err := s.db.Exec(ctx, `
		UPDATE transactions
		SET status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, transactionID, status); err != nil {
		return nil, fmt.Errorf("update qris transaction status: %w", err)
	}

	return &CheckStatusResult{
		TrxID:  trxID,
		Status: status,
	}, nil
}

func (s *Service) findTransactionID(ctx context.Context, tokoID int64, trxID string) (int64, error) {
	const query = `
		SELECT id
		FROM transactions
		WHERE toko_id = $1
			AND category = 'qris'
			AND code = $2
		ORDER BY id
		LIMIT 1
	`

	var id int64
	err := s.db.QueryRow(ctx, query, tokoID, trxID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrTransactionNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("find qris transaction: %w", err)
	}

	return id, nil
}

func normalizeTransactionStatus(status any) string {
	statusText, ok := status.(string)
	if !ok {
		return ""
	}

	switch strings.ToLower(statusText) {
	case "pending":
		return "pending"
	case "success", "paid":
		return "success"
	case "failed":
		return "failed"
	case "expired":
		return "expired"
	default:
		return ""
	}
}

func (p GenerateParams) CustomRefValue() any {
	if p.CustomRef == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*p.CustomRef)
	if trimmed == "" {
		return nil
	}

	return trimmed
}
