package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/jobs"
)

const (
	relayTimeout     = 10 * time.Second
	relayRetryFirst  = 250 * time.Millisecond
	relayRetrySecond = 750 * time.Millisecond
)

type relayScheduler interface {
	EnqueueRelayTokoCallback(ctx context.Context, payload jobs.TokoCallbackPayload) error
}

type Service struct {
	db             *pgxpool.Pool
	logger         zerolog.Logger
	relayScheduler relayScheduler
	httpClient     *http.Client
	notifications  notificationWriter
}

type notificationWriter interface {
	NotifyDepositSuccess(ctx context.Context, ownerUserID int64, amount int64, isNexusggrTopup bool, transactionCode *string) error
	NotifyWithdrawalStatusUpdated(ctx context.Context, ownerUserID int64, amount int64, status string, transactionCode *string) error
	NotifyCallbackDeliveryFailed(ctx context.Context, eventType string, reference string, callbackURL string, statusCode int, failure string) error
}

type qrisTransaction struct {
	ID          int64
	TokoID      int64
	Status      string
	Amount      int64
	Note        *string
	CallbackURL *string
}

func NewService(db *pgxpool.Pool, logger zerolog.Logger, relay relayScheduler) *Service {
	return &Service{
		db:             db,
		logger:         logger.With().Str("module", "webhooks").Logger(),
		relayScheduler: relay,
		httpClient: &http.Client{
			Timeout: relayTimeout,
		},
	}
}

func (s *Service) WithNotifications(service notificationWriter) *Service {
	s.notifications = service
	return s
}

func (s *Service) ProcessQRISCallback(ctx context.Context, payload jobs.QRISCallbackPayload) error {
	transaction, err := s.findTransactionByCode(ctx, payload.TrxID, "deposit")
	if err != nil {
		return err
	}
	if transaction == nil {
		s.logger.Warn().
			Str("trx_id", payload.TrxID).
			Msg("webhook qris transaction not found")
		return nil
	}
	if transaction.Status != "pending" {
		return nil
	}

	sanitizedPayload := sanitizeQRISPayload(payload)
	newStatus := normalizeQRISCallbackStatus(payload.Status)
	if newStatus == "" {
		return fmt.Errorf("unsupported qris callback status %q", payload.Status)
	}

	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		locked, err := s.findTransactionForUpdate(ctx, tx, transaction.ID)
		if err != nil {
			return err
		}
		if locked == nil || locked.Status != "pending" {
			return nil
		}

		noteData := decodeNoteMap(locked.Note)
		isNexusggrTopup := strings.EqualFold(stringValue(noteData["purpose"]), "nexusggr_topup")
		noteJSON, err := json.Marshal(buildQRISNote(noteData, payload, isNexusggrTopup))
		if err != nil {
			return fmt.Errorf("marshal qris note: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE transactions
			SET status = $2,
				amount = $3,
				player = $4,
				note = $5,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, locked.ID, newStatus, payload.Amount, payload.TerminalID, string(noteJSON)); err != nil {
			return fmt.Errorf("update qris transaction: %w", err)
		}

		if newStatus != "success" {
			return nil
		}

		income, err := lockIncome(ctx, tx, false)
		if err != nil {
			return err
		}

		balanceID, err := lockOrCreateBalance(ctx, tx, locked.TokoID)
		if err != nil {
			return err
		}

		if isNexusggrTopup {
			nexusDelta, incomeDelta := calculateNexusTopupAmounts(payload.Amount, income.GGR)
			if _, err := tx.Exec(ctx, `
				UPDATE balances
				SET nexusggr = nexusggr + $2, updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, balanceID, nexusDelta); err != nil {
				return fmt.Errorf("increment nexusggr balance: %w", err)
			}

			if _, err := tx.Exec(ctx, `
				UPDATE incomes
				SET amount = amount + $2, updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, income.ID, incomeDelta); err != nil {
				return fmt.Errorf("increment income amount: %w", err)
			}
		} else {
			pendingDelta, incomeDelta := calculateRegularQRISAmounts(payload.Amount, income.FeeTransaction)
			if _, err := tx.Exec(ctx, `
				UPDATE balances
				SET pending = pending + $2, updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, balanceID, pendingDelta); err != nil {
				return fmt.Errorf("increment pending balance: %w", err)
			}

			if _, err := tx.Exec(ctx, `
				UPDATE incomes
				SET amount = amount + $2, updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, income.ID, incomeDelta); err != nil {
				return fmt.Errorf("increment income amount: %w", err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	s.logger.Info().
		Str("trx_id", payload.TrxID).
		Str("status", newStatus).
		Int64("amount", payload.Amount).
		Int64("toko_id", transaction.TokoID).
		Msg("webhook qris processed")

	if transaction.CallbackURL != nil {
		if err := s.scheduleOrRelayTokoCallback(ctx, jobs.TokoCallbackPayload{
			CallbackURL: transaction.CallbackURL,
			Payload:     sanitizedPayload,
			EventType:   "qris",
			Reference:   payload.TrxID,
		}); err != nil {
			return fmt.Errorf("relay toko qris callback: %w", err)
		}
	}

	if newStatus == "success" && s.notifications != nil {
		ownerUserID, err := s.findTokoOwnerUserID(ctx, transaction.TokoID)
		if err == nil && ownerUserID > 0 {
			code := payload.TrxID
			_ = s.notifications.NotifyDepositSuccess(ctx, ownerUserID, payload.Amount, strings.EqualFold(stringValue(decodeNoteMap(transaction.Note)["purpose"]), "nexusggr_topup"), &code)
		}
	}

	return nil
}

func (s *Service) ProcessDisbursementCallback(ctx context.Context, payload jobs.DisbursementCallbackPayload) error {
	transaction, err := s.findTransactionByCode(ctx, payload.PartnerRefNo, "withdrawal")
	if err != nil {
		return err
	}
	if transaction == nil {
		s.logger.Warn().
			Str("partner_ref_no", payload.PartnerRefNo).
			Msg("webhook disbursement transaction not found")
		return nil
	}
	if transaction.Status != "pending" {
		return nil
	}

	newStatus := strings.ToLower(strings.TrimSpace(payload.Status))
	sanitizedPayload := sanitizeDisbursementPayload(payload)

	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		locked, err := s.findTransactionForUpdate(ctx, tx, transaction.ID)
		if err != nil {
			return err
		}
		if locked == nil || locked.Status != "pending" {
			return nil
		}

		noteData := decodeNoteMap(locked.Note)
		noteData["transaction_date"] = payload.TransactionDate

		noteJSON, err := json.Marshal(noteData)
		if err != nil {
			return fmt.Errorf("marshal disbursement note: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE transactions
			SET status = $2,
				note = $3,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, locked.ID, newStatus, string(noteJSON)); err != nil {
			return fmt.Errorf("update disbursement transaction: %w", err)
		}

		platformFee := intValue(noteData["platform_fee"])
		bankFee := intValue(noteData["fee"])

		if newStatus == "success" {
			income, err := lockIncome(ctx, tx, true)
			if err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `
				UPDATE incomes
				SET amount = amount + $2, updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, income.ID, platformFee); err != nil {
				return fmt.Errorf("increment withdrawal income: %w", err)
			}

			return nil
		}

		balanceID, err := lockOrCreateBalance(ctx, tx, locked.TokoID)
		if err != nil {
			return err
		}

		refundAmount := calculateDisbursementRefund(locked.Amount, platformFee, bankFee)
		if _, err := tx.Exec(ctx, `
			UPDATE balances
			SET settle = settle + $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, balanceID, refundAmount); err != nil {
			return fmt.Errorf("refund settle balance: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	s.logger.Info().
		Str("partner_ref_no", payload.PartnerRefNo).
		Str("status", newStatus).
		Int64("amount", payload.Amount).
		Int64("toko_id", transaction.TokoID).
		Msg("webhook disbursement processed")

	if transaction.CallbackURL != nil {
		if err := s.scheduleOrRelayTokoCallback(ctx, jobs.TokoCallbackPayload{
			CallbackURL: transaction.CallbackURL,
			Payload:     sanitizedPayload,
			EventType:   "disbursement",
			Reference:   payload.PartnerRefNo,
		}); err != nil {
			return fmt.Errorf("relay toko disbursement callback: %w", err)
		}
	}

	if s.notifications != nil {
		ownerUserID, err := s.findTokoOwnerUserID(ctx, transaction.TokoID)
		if err == nil && ownerUserID > 0 {
			code := payload.PartnerRefNo
			_ = s.notifications.NotifyWithdrawalStatusUpdated(ctx, ownerUserID, payload.Amount, newStatus, &code)
		}
	}

	return nil
}

func (s *Service) scheduleOrRelayTokoCallback(ctx context.Context, payload jobs.TokoCallbackPayload) error {
	if s.relayScheduler != nil {
		if err := s.relayScheduler.EnqueueRelayTokoCallback(ctx, payload); err == nil {
			return nil
		} else {
			s.logger.Error().
				Err(err).
				Str("event_type", payload.EventType).
				Str("reference", payload.Reference).
				Msg("failed to enqueue toko callback, falling back to direct relay")
		}
	}

	return s.RelayTokoCallback(ctx, payload)
}

func (s *Service) RelayTokoCallback(ctx context.Context, payload jobs.TokoCallbackPayload) error {
	if payload.CallbackURL == nil || strings.TrimSpace(*payload.CallbackURL) == "" {
		s.logger.Info().
			Str("event_type", payload.EventType).
			Str("reference", payload.Reference).
			Msg("toko callback skipped because callback_url is empty")
		return nil
	}

	callbackURL := strings.TrimSpace(*payload.CallbackURL)
	if _, err := url.ParseRequestURI(callbackURL); err != nil {
		s.logger.Warn().
			Str("event_type", payload.EventType).
			Str("reference", payload.Reference).
			Str("callback_url", callbackURL).
			Msg("toko callback skipped because callback_url is invalid")
		if s.notifications != nil {
			_ = s.notifications.NotifyCallbackDeliveryFailed(ctx, payload.EventType, payload.Reference, callbackURL, 0, "invalid callback_url")
		}
		return nil
	}

	body, err := json.Marshal(payload.Payload)
	if err != nil {
		return fmt.Errorf("marshal toko callback payload: %w", err)
	}

	attempts := []time.Duration{0, relayRetryFirst, relayRetrySecond}
	var lastErr error
	for idx, delay := range attempts {
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build toko callback request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			s.logger.Info().
				Str("event_type", payload.EventType).
				Str("reference", payload.Reference).
				Str("callback_url", callbackURL).
				Int("attempt", idx+1).
				Msg("toko callback delivered")
			return nil
		}

		lastErr = fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	s.logger.Warn().
		Str("event_type", payload.EventType).
		Str("reference", payload.Reference).
		Str("callback_url", callbackURL).
		Err(lastErr).
		Msg("toko callback delivery failed")

	if s.notifications != nil {
		_ = s.notifications.NotifyCallbackDeliveryFailed(ctx, payload.EventType, payload.Reference, callbackURL, 0, lastErr.Error())
	}

	return fmt.Errorf("deliver toko callback: %w", lastErr)
}

type incomeRecord struct {
	ID             int64
	GGR            int64
	FeeTransaction int64
}

func (s *Service) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	committed = true
	return nil
}

func (s *Service) findTransactionByCode(ctx context.Context, code string, transactionType string) (*qrisTransaction, error) {
	const query = `
		SELECT
			t.id,
			t.toko_id,
			t.status,
			t.amount::bigint,
			t.note,
			k.callback_url
		FROM transactions t
		LEFT JOIN tokos k ON k.id = t.toko_id
		WHERE t.code = $1
			AND t.category = 'qris'
			AND t.type = $2
		ORDER BY t.id
		LIMIT 1
	`

	var transaction qrisTransaction
	err := s.db.QueryRow(ctx, query, code, transactionType).Scan(
		&transaction.ID,
		&transaction.TokoID,
		&transaction.Status,
		&transaction.Amount,
		&transaction.Note,
		&transaction.CallbackURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find transaction by code: %w", err)
	}

	return &transaction, nil
}

func (s *Service) findTransactionForUpdate(ctx context.Context, tx pgx.Tx, transactionID int64) (*qrisTransaction, error) {
	const query = `
		SELECT id, toko_id, status, amount::bigint, note
		FROM transactions
		WHERE id = $1
		FOR UPDATE
	`

	var transaction qrisTransaction
	err := tx.QueryRow(ctx, query, transactionID).Scan(
		&transaction.ID,
		&transaction.TokoID,
		&transaction.Status,
		&transaction.Amount,
		&transaction.Note,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lock transaction: %w", err)
	}

	return &transaction, nil
}

func lockIncome(ctx context.Context, tx pgx.Tx, createIfMissing bool) (*incomeRecord, error) {
	const selectQuery = `
		SELECT id, ggr::bigint, fee_transaction::bigint
		FROM incomes
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`

	var income incomeRecord
	err := tx.QueryRow(ctx, selectQuery).Scan(&income.ID, &income.GGR, &income.FeeTransaction)
	if err == nil {
		return &income, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lock income: %w", err)
	}
	if !createIfMissing {
		return nil, fmt.Errorf("income row is required before processing callback")
	}

	const insertQuery = `
		INSERT INTO incomes (ggr, fee_transaction, fee_withdrawal, amount, created_at, updated_at)
		VALUES (7, 3, 15, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, ggr::bigint, fee_transaction::bigint
	`

	if err := tx.QueryRow(ctx, insertQuery).Scan(&income.ID, &income.GGR, &income.FeeTransaction); err != nil {
		return nil, fmt.Errorf("create income row: %w", err)
	}

	return &income, nil
}

func lockOrCreateBalance(ctx context.Context, tx pgx.Tx, tokoID int64) (int64, error) {
	const selectQuery = `
		SELECT id
		FROM balances
		WHERE toko_id = $1
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`

	var balanceID int64
	err := tx.QueryRow(ctx, selectQuery, tokoID).Scan(&balanceID)
	if err == nil {
		return balanceID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("lock balance: %w", err)
	}

	const insertQuery = `
		INSERT INTO balances (toko_id, settle, pending, nexusggr, created_at, updated_at)
		VALUES ($1, 0, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	if err := tx.QueryRow(ctx, insertQuery, tokoID).Scan(&balanceID); err != nil {
		return 0, fmt.Errorf("create balance row: %w", err)
	}

	return balanceID, nil
}

func sanitizeQRISPayload(payload jobs.QRISCallbackPayload) map[string]any {
	return map[string]any{
		"amount":      payload.Amount,
		"terminal_id": payload.TerminalID,
		"trx_id":      payload.TrxID,
		"rrn":         payload.RRN,
		"custom_ref":  payload.CustomRef,
		"vendor":      payload.Vendor,
		"status":      payload.Status,
		"created_at":  payload.CreatedAt,
		"finish_at":   payload.FinishAt,
	}
}

func sanitizeDisbursementPayload(payload jobs.DisbursementCallbackPayload) map[string]any {
	return map[string]any{
		"amount":           payload.Amount,
		"partner_ref_no":   payload.PartnerRefNo,
		"status":           payload.Status,
		"transaction_date": payload.TransactionDate,
	}
}

func normalizeQRISCallbackStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
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

func buildQRISNote(existing map[string]any, payload jobs.QRISCallbackPayload, isNexusggrTopup bool) map[string]any {
	if isNexusggrTopup {
		note := cloneMap(existing)
		note["rrn"] = payload.RRN
		note["vendor"] = payload.Vendor
		note["finish_at"] = payload.FinishAt
		return note
	}

	return map[string]any{
		"rrn":        payload.RRN,
		"vendor":     payload.Vendor,
		"custom_ref": payload.CustomRef,
		"finish_at":  payload.FinishAt,
	}
}

func calculateRegularQRISAmounts(amount int64, feeTransaction int64) (int64, int64) {
	plusIncome := float64(amount) * float64(feeTransaction) / 100
	finalPending := float64(amount) - plusIncome
	return int64(math.Round(finalPending)), int64(math.Round(plusIncome))
}

func calculateNexusTopupAmounts(amount int64, ggr int64) (int64, int64) {
	if ggr <= 0 {
		return 0, amount - 1800
	}

	nexus := float64(amount) * 100 / float64(ggr)
	return int64(math.Round(nexus)), amount - 1800
}

func calculateDisbursementRefund(amount int64, platformFee int64, bankFee int64) int64 {
	return amount + platformFee + bankFee
}

func (s *Service) findTokoOwnerUserID(ctx context.Context, tokoID int64) (int64, error) {
	var userID int64
	if err := s.db.QueryRow(ctx, `
		SELECT user_id
		FROM tokos
		WHERE id = $1
		LIMIT 1
	`, tokoID).Scan(&userID); err != nil {
		return 0, fmt.Errorf("find toko owner user id: %w", err)
	}

	return userID, nil
}

func decodeNoteMap(note *string) map[string]any {
	if note == nil || strings.TrimSpace(*note) == "" {
		return map[string]any{}
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(*note), &decoded); err != nil {
		return map[string]any{}
	}

	return decoded
}

func cloneMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func intValue(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(math.Round(typed))
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed
	case string:
		number := json.Number(strings.TrimSpace(typed))
		parsed, _ := number.Int64()
		return parsed
	default:
		return 0
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case *string:
		if typed == nil {
			return ""
		}
		return *typed
	default:
		return ""
	}
}
