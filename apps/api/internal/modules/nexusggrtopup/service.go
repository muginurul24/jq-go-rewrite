package nexusggrtopup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	qrisintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/qris"
)

const (
	minimumTopupAmount  int64 = 1000
	defaultExpireSecond int   = 300
	defaultTopupRatio   int64 = 7
	discountedTopupRatio int64 = 6
	discountedTopupThreshold int64 = 1_000_000
	topupPurpose              = "nexusggr_topup"
)

var (
	ErrTokoNotFound             = errors.New("toko not found")
	ErrTopupAmountTooSmall      = errors.New("topup amount too small")
	ErrGenerateTopupFailed      = errors.New("generate topup failed")
	ErrTopupTransactionNotFound = errors.New("topup transaction not found")
)

type qrisClient interface {
	Generate(ctx context.Context, username string, amount int64, expire int, customRef *string) (*qrisintegration.GenerateResponse, error)
}

type TokoOption struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	OwnerUsername   string `json:"ownerUsername"`
	NexusggrBalance int64  `json:"nexusggrBalance"`
}

type PendingTopup struct {
	Amount          int64  `json:"amount"`
	TransactionCode string `json:"transactionCode"`
	ExpiresAt       *int64 `json:"expiresAt,omitempty"`
	Status          string `json:"status"`
	QrPayload       string `json:"qrPayload"`
}

type TopupRateRule struct {
	ThresholdAmount    int64 `json:"thresholdAmount"`
	BelowThresholdRate int64 `json:"belowThresholdRate"`
	AboveThresholdRate int64 `json:"aboveThresholdRate"`
}

type BootstrapResult struct {
	Tokos        []TokoOption  `json:"tokos"`
	SelectedToko *TokoOption   `json:"selectedToko,omitempty"`
	TopupRatio   int64         `json:"topupRatio"`
	TopupRule    TopupRateRule `json:"topupRule"`
	PendingTopup *PendingTopup `json:"pendingTopup,omitempty"`
}

type GenerateResult struct {
	SelectedToko *TokoOption  `json:"selectedToko,omitempty"`
	TopupRatio   int64        `json:"topupRatio"`
	TopupRule    TopupRateRule `json:"topupRule"`
	PendingTopup PendingTopup `json:"pendingTopup"`
}

type StatusResult struct {
	Status       string        `json:"status"`
	PendingTopup *PendingTopup `json:"pendingTopup,omitempty"`
}

type Service struct {
	db     *pgxpool.Pool
	client qrisClient
}

func NewService(db *pgxpool.Pool, client qrisClient) *Service {
	return &Service{
		db:     db,
		client: client,
	}
}

func (s *Service) Bootstrap(ctx context.Context, actor auth.PublicUser, selectedTokoID *int64) (*BootstrapResult, error) {
	tokos, err := s.accessibleTokos(ctx, actor)
	if err != nil {
		return nil, err
	}

	var selected *TokoOption
	if len(tokos) > 0 {
		selected = &tokos[0]
	}
	if selectedTokoID != nil {
		for _, option := range tokos {
			if option.ID == *selectedTokoID {
				selected = &option
				break
			}
		}
	}

	result := &BootstrapResult{
		Tokos:        tokos,
		SelectedToko: selected,
		TopupRatio:   defaultTopupRatio,
		TopupRule:    DefaultTopupRateRule(),
	}

	if selected != nil {
		pending, err := s.restorePendingTopup(ctx, selected.ID)
		if err != nil {
			return nil, err
		}
		result.PendingTopup = pending
	}

	return result, nil
}

func (s *Service) Generate(ctx context.Context, actor auth.PublicUser, tokoID int64, amount int64) (*GenerateResult, error) {
	if amount < minimumTopupAmount {
		return nil, ErrTopupAmountTooSmall
	}

	toko, err := s.findAccessibleToko(ctx, actor, tokoID)
	if err != nil {
		return nil, err
	}

	upstreamResponse, err := s.client.Generate(ctx, toko.OwnerUsername, amount, defaultExpireSecond, nil)
	if err != nil || !upstreamResponse.Status {
		return nil, ErrGenerateTopupFailed
	}

	trxID, _ := upstreamResponse.TrxID.(string)
	qrPayload, _ := upstreamResponse.Data.(string)
	expiresAt := normalizeExpiry(upstreamResponse.ExpiredAt)

	noteJSON, err := json.Marshal(map[string]any{
		"purpose":    topupPurpose,
		"qris_data":  qrPayload,
		"expired_at": expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal topup note: %w", err)
	}

	if _, err := s.db.Exec(ctx, `
		INSERT INTO transactions (
			toko_id,
			category,
			type,
			status,
			amount,
			code,
			note,
			created_at,
			updated_at
		) VALUES ($1, 'qris', 'deposit', 'pending', $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, toko.ID, amount, trxID, string(noteJSON)); err != nil {
		return nil, fmt.Errorf("insert topup transaction: %w", err)
	}

	ratio := ResolveTopupRatio(amount)

	return &GenerateResult{
		SelectedToko: toko,
		TopupRatio:   ratio,
		TopupRule:    DefaultTopupRateRule(),
		PendingTopup: PendingTopup{
			Amount:          amount,
			TransactionCode: trxID,
			ExpiresAt:       expiresAt,
			Status:          "pending",
			QrPayload:       qrPayload,
		},
	}, nil
}

func (s *Service) CheckStatus(ctx context.Context, actor auth.PublicUser, tokoID int64, transactionCode string) (*StatusResult, error) {
	toko, err := s.findAccessibleToko(ctx, actor, tokoID)
	if err != nil {
		return nil, err
	}

	transaction, err := s.findTopupTransaction(ctx, toko.ID, transactionCode)
	if err != nil {
		return nil, err
	}

	pending, err := pendingTopupFromRecord(transaction)
	if err != nil {
		return nil, err
	}

	if pending.Status == "success" {
		return &StatusResult{Status: "success"}, nil
	}

	if pending.ExpiresAt != nil && time.Now().Unix() >= *pending.ExpiresAt {
		if transaction.Status == "pending" {
			if _, err := s.db.Exec(ctx, `
				UPDATE transactions
				SET status = 'expired', updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, transaction.ID); err != nil {
				return nil, fmt.Errorf("expire topup transaction: %w", err)
			}
		}

		return &StatusResult{Status: "expired"}, nil
	}

	return &StatusResult{
		Status:       transaction.Status,
		PendingTopup: pending,
	}, nil
}

type topupTransaction struct {
	ID     int64
	Status string
	Amount int64
	Code   string
	Note   *string
}

func (s *Service) accessibleTokos(ctx context.Context, actor auth.PublicUser) ([]TokoOption, error) {
	query := `
		SELECT
			t.id,
			t.name,
			u.username,
			COALESCE(b.nexusggr, 0) AS nexusggr_balance
		FROM tokos t
		INNER JOIN users u ON u.id = t.user_id
		LEFT JOIN balances b ON b.toko_id = t.id
		WHERE t.deleted_at IS NULL
			AND t.is_active = TRUE
	`
	args := []any{}

	if actor.Role != "dev" && actor.Role != "superadmin" {
		query += " AND t.user_id = $1"
		args = append(args, actor.ID)
	}

	query += " ORDER BY t.name ASC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list accessible topup tokos: %w", err)
	}
	defer rows.Close()

	options := make([]TokoOption, 0, 8)
	for rows.Next() {
		var option TokoOption
		if err := rows.Scan(&option.ID, &option.Name, &option.OwnerUsername, &option.NexusggrBalance); err != nil {
			return nil, fmt.Errorf("scan accessible topup toko: %w", err)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accessible topup tokos: %w", err)
	}

	return options, nil
}

func (s *Service) findAccessibleToko(ctx context.Context, actor auth.PublicUser, tokoID int64) (*TokoOption, error) {
	tokos, err := s.accessibleTokos(ctx, actor)
	if err != nil {
		return nil, err
	}

	for _, option := range tokos {
		if option.ID == tokoID {
			item := option
			return &item, nil
		}
	}

	return nil, ErrTokoNotFound
}

func (s *Service) restorePendingTopup(ctx context.Context, tokoID int64) (*PendingTopup, error) {
	transaction, err := s.latestPendingTopup(ctx, tokoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	pending, err := pendingTopupFromRecord(transaction)
	if err != nil {
		return nil, err
	}

	if pending.ExpiresAt != nil && time.Now().Unix() >= *pending.ExpiresAt {
		if _, err := s.db.Exec(ctx, `
			UPDATE transactions
			SET status = 'expired', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, transaction.ID); err != nil {
			return nil, fmt.Errorf("expire restored topup transaction: %w", err)
		}

		return nil, nil
	}

	return pending, nil
}

func DefaultTopupRateRule() TopupRateRule {
	return TopupRateRule{
		ThresholdAmount:    discountedTopupThreshold,
		BelowThresholdRate: defaultTopupRatio,
		AboveThresholdRate: discountedTopupRatio,
	}
}

func ResolveTopupRatio(amount int64) int64 {
	if amount > discountedTopupThreshold {
		return discountedTopupRatio
	}
	return defaultTopupRatio
}

func (s *Service) latestPendingTopup(ctx context.Context, tokoID int64) (*topupTransaction, error) {
	query := `
		SELECT id, status, amount::bigint, code, note
		FROM transactions
		WHERE toko_id = $1
			AND category = 'qris'
			AND type = 'deposit'
			AND status = 'pending'
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`

	var item topupTransaction
	if err := s.db.QueryRow(ctx, query, tokoID).Scan(
		&item.ID,
		&item.Status,
		&item.Amount,
		&item.Code,
		&item.Note,
	); err != nil {
		return nil, err
	}

	note, _ := decodeNote(item.Note)
	if strings.TrimSpace(stringValue(note["purpose"])) != topupPurpose {
		return nil, pgx.ErrNoRows
	}

	return &item, nil
}

func (s *Service) findTopupTransaction(ctx context.Context, tokoID int64, transactionCode string) (*topupTransaction, error) {
	query := `
		SELECT id, status, amount::bigint, code, note
		FROM transactions
		WHERE toko_id = $1
			AND category = 'qris'
			AND type = 'deposit'
			AND code = $2
		ORDER BY id DESC
		LIMIT 1
	`

	var item topupTransaction
	if err := s.db.QueryRow(ctx, query, tokoID, strings.TrimSpace(transactionCode)).Scan(
		&item.ID,
		&item.Status,
		&item.Amount,
		&item.Code,
		&item.Note,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTopupTransactionNotFound
		}
		return nil, fmt.Errorf("find topup transaction: %w", err)
	}

	note, _ := decodeNote(item.Note)
	if strings.TrimSpace(stringValue(note["purpose"])) != topupPurpose {
		return nil, ErrTopupTransactionNotFound
	}

	return &item, nil
}

func pendingTopupFromRecord(record *topupTransaction) (*PendingTopup, error) {
	note, err := decodeNote(record.Note)
	if err != nil {
		return nil, err
	}

	qrPayload := stringValue(note["qris_data"])
	if strings.TrimSpace(qrPayload) == "" {
		return nil, fmt.Errorf("pending topup qris payload missing")
	}

	expiresAt := int64Value(note["expired_at"])
	return &PendingTopup{
		Amount:          record.Amount,
		TransactionCode: record.Code,
		ExpiresAt:       expiresAt,
		Status:          record.Status,
		QrPayload:       qrPayload,
	}, nil
}

func decodeNote(note *string) (map[string]any, error) {
	if note == nil || strings.TrimSpace(*note) == "" {
		return map[string]any{}, nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(*note), &payload); err != nil {
		return nil, fmt.Errorf("decode topup note: %w", err)
	}

	return payload, nil
}

func stringValue(value any) string {
	typed, _ := value.(string)
	return typed
}

func int64Value(value any) *int64 {
	switch typed := value.(type) {
	case float64:
		result := int64(typed)
		return &result
	case int64:
		return &typed
	}

	return nil
}

func normalizeExpiry(value any) *int64 {
	switch typed := value.(type) {
	case int64:
		if typed < 1_000_000_000 {
			normalized := time.Now().Unix() + typed
			return &normalized
		}
		return &typed
	case float64:
		parsed := int64(typed)
		if parsed < 1_000_000_000 {
			normalized := time.Now().Unix() + parsed
			return &normalized
		}
		return &parsed
	}

	return nil
}
