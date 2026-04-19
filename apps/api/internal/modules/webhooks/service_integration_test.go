package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/jobs"
)

var (
	integrationHarnessOnce sync.Once
	integrationHarness     *webhookIntegrationHarness
	integrationHarnessErr  error
)

type webhookIntegrationHarness struct {
	db   *embeddedpostgres.EmbeddedPostgres
	pool *pgxpool.Pool
}

func TestMain(m *testing.M) {
	code := m.Run()

	if integrationHarness != nil {
		integrationHarness.close()
	}

	os.Exit(code)
}

func TestProcessQRISCallbackRegularDepositIntegration(t *testing.T) {
	harness := requireWebhookIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	harness.seedUserAndToko(t, ctx, 11, 21, 0)
	harness.seedIncome(t, ctx, 7, 3, 15, 0)
	harness.seedTransaction(t, ctx, 31, 21, "qris", "deposit", "pending", 100000, "trx-regular-001", `{"purpose":"generate","expired_at":1713330000}`)

	service := NewService(harness.pool, zerolog.Nop(), nil)

	customRef := "REF-001"
	rrn := "RRN-001"
	vendor := "qris"
	finishAt := "2026-04-17T12:30:00+07:00"
	if err := service.ProcessQRISCallback(ctx, jobs.QRISCallbackPayload{
		Amount:     100000,
		TerminalID: "terminal-a",
		TrxID:      "trx-regular-001",
		RRN:        &rrn,
		CustomRef:  &customRef,
		Vendor:     &vendor,
		Status:     "success",
		FinishAt:   &finishAt,
	}); err != nil {
		t.Fatalf("process qris callback: %v", err)
	}

	transaction := harness.fetchTransaction(t, ctx, "trx-regular-001")
	if transaction.Status != "success" {
		t.Fatalf("expected transaction status success, got %s", transaction.Status)
	}
	if transaction.Player == nil || *transaction.Player != "terminal-a" {
		t.Fatalf("expected transaction player terminal-a, got %#v", transaction.Player)
	}

	note := decodeJSONMap(t, transaction.Note)
	if _, exists := note["expired_at"]; exists {
		t.Fatalf("expected regular deposit note to drop expired_at, got %#v", note)
	}
	if note["custom_ref"] != customRef {
		t.Fatalf("expected custom_ref %q, got %#v", customRef, note["custom_ref"])
	}

	balance := harness.fetchBalance(t, ctx, 21)
	if balance.Pending != 97000 {
		t.Fatalf("expected pending balance 97000, got %d", balance.Pending)
	}
	if balance.Nexusggr != 0 {
		t.Fatalf("expected nexusggr balance unchanged, got %d", balance.Nexusggr)
	}

	income := harness.fetchIncome(t, ctx)
	if income.Amount != 3000 {
		t.Fatalf("expected income amount 3000, got %d", income.Amount)
	}
}

func TestProcessQRISCallbackNexusTopupIntegration(t *testing.T) {
	harness := requireWebhookIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	harness.seedUserAndToko(t, ctx, 12, 22, 0)
	harness.seedIncome(t, ctx, 7, 3, 15, 0)
	harness.seedTransaction(t, ctx, 32, 22, "qris", "deposit", "pending", 100000, "trx-topup-001", `{"purpose":"nexusggr_topup","qris_data":"000201","expired_at":1713330000}`)

	service := NewService(harness.pool, zerolog.Nop(), nil)

	rrn := "RRN-002"
	vendor := "qris"
	finishAt := "2026-04-17T12:31:00+07:00"
	if err := service.ProcessQRISCallback(ctx, jobs.QRISCallbackPayload{
		Amount:     100000,
		TerminalID: "terminal-topup",
		TrxID:      "trx-topup-001",
		RRN:        &rrn,
		Vendor:     &vendor,
		Status:     "success",
		FinishAt:   &finishAt,
	}); err != nil {
		t.Fatalf("process qris topup callback: %v", err)
	}

	balance := harness.fetchBalance(t, ctx, 22)
	if balance.Nexusggr != 1428571 {
		t.Fatalf("expected nexusggr balance 1428571, got %d", balance.Nexusggr)
	}
	if balance.Pending != 0 {
		t.Fatalf("expected pending balance unchanged, got %d", balance.Pending)
	}

	income := harness.fetchIncome(t, ctx)
	if income.Amount != 98200 {
		t.Fatalf("expected income amount 98200, got %d", income.Amount)
	}

	transaction := harness.fetchTransaction(t, ctx, "trx-topup-001")
	note := decodeJSONMap(t, transaction.Note)
	if note["qris_data"] != "000201" {
		t.Fatalf("expected qris_data to be preserved, got %#v", note["qris_data"])
	}
	if note["expired_at"] == nil {
		t.Fatalf("expected expired_at to remain in note, got %#v", note)
	}
}

func TestProcessQRISCallbackNexusTopupUsesDiscountedRatioAboveThresholdIntegration(t *testing.T) {
	harness := requireWebhookIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	harness.seedUserAndToko(t, ctx, 13, 23, 0)
	harness.seedIncome(t, ctx, 7, 3, 15, 0)
	harness.seedTransaction(t, ctx, 36, 23, "qris", "deposit", "pending", 1_500_000, "trx-topup-002", `{"purpose":"nexusggr_topup","qris_data":"000201","expired_at":1713330000}`)

	service := NewService(harness.pool, zerolog.Nop(), nil)

	rrn := "RRN-003"
	vendor := "qris"
	finishAt := "2026-04-17T12:32:00+07:00"
	if err := service.ProcessQRISCallback(ctx, jobs.QRISCallbackPayload{
		Amount:     1_500_000,
		TerminalID: "terminal-topup-discount",
		TrxID:      "trx-topup-002",
		RRN:        &rrn,
		Vendor:     &vendor,
		Status:     "success",
		FinishAt:   &finishAt,
	}); err != nil {
		t.Fatalf("process discounted qris topup callback: %v", err)
	}

	balance := harness.fetchBalance(t, ctx, 23)
	if balance.Nexusggr != 25_000_000 {
		t.Fatalf("expected discounted nexusggr balance 25000000, got %d", balance.Nexusggr)
	}

	income := harness.fetchIncome(t, ctx)
	if income.Amount != 1_498_200 {
		t.Fatalf("expected income amount 1498200, got %d", income.Amount)
	}
}

func TestProcessQRISCallbackNonSuccessUpdatesStatusWithoutBalanceSideEffectsIntegration(t *testing.T) {
	harness := requireWebhookIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	harness.seedUserAndToko(t, ctx, 15, 25, 0)
	harness.seedIncome(t, ctx, 7, 3, 15, 0)
	harness.setTokoCallbackURL(t, ctx, 25, "https://callback-store.test/qris")
	harness.seedTransaction(t, ctx, 35, 25, "qris", "deposit", "pending", 200000, "trx-failed-001", `{"purpose":"generate","custom_ref":"REF-FAILED"}`)

	relay := &capturingRelayScheduler{}
	service := NewService(harness.pool, zerolog.Nop(), relay)

	rrn := "RRN-FAILED"
	if err := service.ProcessQRISCallback(ctx, jobs.QRISCallbackPayload{
		Amount:     200000,
		TerminalID: "terminal-failed",
		TrxID:      "trx-failed-001",
		RRN:        &rrn,
		Status:     "failed",
	}); err != nil {
		t.Fatalf("process qris failed callback: %v", err)
	}

	transaction := harness.fetchTransaction(t, ctx, "trx-failed-001")
	if transaction.Status != "failed" {
		t.Fatalf("expected transaction status failed, got %s", transaction.Status)
	}

	balance := harness.fetchBalance(t, ctx, 25)
	if balance.Pending != 0 || balance.Nexusggr != 0 {
		t.Fatalf("expected no balance side effects, got pending=%d nexusggr=%d", balance.Pending, balance.Nexusggr)
	}

	income := harness.fetchIncome(t, ctx)
	if income.Amount != 0 {
		t.Fatalf("expected income to remain 0 for failed callback, got %d", income.Amount)
	}

	if len(relay.payloads) != 1 {
		t.Fatalf("expected exactly one relay payload, got %d", len(relay.payloads))
	}
	if status, _ := relay.payloads[0].Payload["status"].(string); status != "failed" {
		t.Fatalf("expected relay payload status failed, got %#v", relay.payloads[0].Payload["status"])
	}
}

func TestProcessDisbursementCallbackSuccessIntegration(t *testing.T) {
	harness := requireWebhookIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	harness.seedUserAndToko(t, ctx, 13, 23, 200000)
	harness.seedIncome(t, ctx, 7, 3, 15, 1000)
	harness.seedTransaction(t, ctx, 33, 23, "qris", "withdrawal", "pending", 50000, "wd-success-001", `{"purpose":"withdrawal","platform_fee":7500,"fee":6500}`)

	service := NewService(harness.pool, zerolog.Nop(), nil)

	if err := service.ProcessDisbursementCallback(ctx, jobs.DisbursementCallbackPayload{
		Amount:          50000,
		PartnerRefNo:    "wd-success-001",
		Status:          "success",
		TransactionDate: stringPointer("2026-04-17 12:45:00"),
	}); err != nil {
		t.Fatalf("process disbursement success callback: %v", err)
	}

	transaction := harness.fetchTransaction(t, ctx, "wd-success-001")
	if transaction.Status != "success" {
		t.Fatalf("expected withdrawal status success, got %s", transaction.Status)
	}
	note := decodeJSONMap(t, transaction.Note)
	if note["transaction_date"] != "2026-04-17 12:45:00" {
		t.Fatalf("expected transaction_date to be stored, got %#v", note["transaction_date"])
	}

	balance := harness.fetchBalance(t, ctx, 23)
	if balance.Settle != 200000 {
		t.Fatalf("expected settle balance unchanged on success, got %d", balance.Settle)
	}

	income := harness.fetchIncome(t, ctx)
	if income.Amount != 8500 {
		t.Fatalf("expected income amount 8500, got %d", income.Amount)
	}
}

func TestProcessDisbursementCallbackFailureRefundsSettleIntegration(t *testing.T) {
	harness := requireWebhookIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	harness.seedUserAndToko(t, ctx, 14, 24, 200000)
	harness.seedIncome(t, ctx, 7, 3, 15, 1000)
	harness.seedTransaction(t, ctx, 34, 24, "qris", "withdrawal", "pending", 50000, "wd-failed-001", `{"purpose":"withdrawal","platform_fee":7500,"fee":6500}`)

	service := NewService(harness.pool, zerolog.Nop(), nil)

	if err := service.ProcessDisbursementCallback(ctx, jobs.DisbursementCallbackPayload{
		Amount:          50000,
		PartnerRefNo:    "wd-failed-001",
		Status:          "failed",
		TransactionDate: stringPointer("2026-04-17 12:50:00"),
	}); err != nil {
		t.Fatalf("process disbursement failed callback: %v", err)
	}

	transaction := harness.fetchTransaction(t, ctx, "wd-failed-001")
	if transaction.Status != "failed" {
		t.Fatalf("expected withdrawal status failed, got %s", transaction.Status)
	}

	balance := harness.fetchBalance(t, ctx, 24)
	if balance.Settle != 264000 {
		t.Fatalf("expected settle balance refunded to 264000, got %d", balance.Settle)
	}

	income := harness.fetchIncome(t, ctx)
	if income.Amount != 1000 {
		t.Fatalf("expected income amount unchanged on failure, got %d", income.Amount)
	}
}

func requireWebhookIntegrationHarness(t *testing.T) *webhookIntegrationHarness {
	t.Helper()

	integrationHarnessOnce.Do(func() {
		integrationHarness, integrationHarnessErr = startWebhookIntegrationHarness()
	})

	if integrationHarnessErr != nil {
		t.Skipf("skip webhook integration harness: %v", integrationHarnessErr)
	}

	return integrationHarness
}

func startWebhookIntegrationHarness() (*webhookIntegrationHarness, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}

	basePath, err := os.MkdirTemp("", "justqiuv2-webhooks-pg-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	config := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(uint32(port)).
		Database("justqiuv2_webhooks_test").
		Username("postgres").
		Password("postgres").
		BinariesPath(filepath.Join(basePath, "bin")).
		DataPath(filepath.Join(basePath, "data")).
		RuntimePath(filepath.Join(basePath, "run")).
		CachePath(filepath.Join(basePath, "cache")).
		StartTimeout(2 * time.Minute)

	db := embeddedpostgres.NewDatabase(config)
	if err := db.Start(); err != nil {
		return nil, fmt.Errorf("start embedded postgres: %w", err)
	}

	pool, err := pgxpool.New(context.Background(), config.GetConnectionURL())
	if err != nil {
		_ = db.Stop()
		return nil, fmt.Errorf("connect embedded postgres: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		_ = db.Stop()
		return nil, fmt.Errorf("ping embedded postgres: %w", err)
	}

	if err := applyWebhookIntegrationSchema(context.Background(), pool); err != nil {
		pool.Close()
		_ = db.Stop()
		return nil, err
	}

	return &webhookIntegrationHarness{
		db:   db,
		pool: pool,
	}, nil
}

func applyWebhookIntegrationSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("resolve current file path")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "00001_legacy_core.sql")
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	upSection := string(content)
	if marker := strings.Index(upSection, "-- +goose Down"); marker >= 0 {
		upSection = upSection[:marker]
	}
	upSection = strings.ReplaceAll(upSection, "-- +goose Up", "")

	if _, err := pool.Exec(ctx, upSection); err != nil {
		return fmt.Errorf("apply integration schema: %w", err)
	}

	return nil
}

func (h *webhookIntegrationHarness) reset(t *testing.T) {
	t.Helper()

	if _, err := h.pool.Exec(context.Background(), `
		TRUNCATE TABLE
			players,
			transactions,
			incomes,
			balances,
			banks,
			tokos,
			notifications,
			personal_access_tokens,
			sessions,
			password_reset_tokens,
			users
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("reset integration tables: %v", err)
	}
}

func (h *webhookIntegrationHarness) close() {
	if h == nil {
		return
	}
	if h.pool != nil {
		h.pool.Close()
	}
	if h.db != nil {
		_ = h.db.Stop()
	}
}

func (h *webhookIntegrationHarness) seedUserAndToko(t *testing.T, ctx context.Context, userID int64, tokoID int64, settle int64) {
	t.Helper()

	if _, err := h.pool.Exec(ctx, `
		INSERT INTO users (id, username, name, email, password, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'hashed-password', 'admin', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, userID, fmt.Sprintf("user-%d", userID), fmt.Sprintf("User %d", userID), fmt.Sprintf("user-%d@example.test", userID)); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	if _, err := h.pool.Exec(ctx, `
		INSERT INTO tokos (id, user_id, name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, tokoID, userID, fmt.Sprintf("Toko %d", tokoID)); err != nil {
		t.Fatalf("seed toko: %v", err)
	}

	if _, err := h.pool.Exec(ctx, `
		INSERT INTO balances (toko_id, settle, pending, nexusggr, created_at, updated_at)
		VALUES ($1, $2, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, tokoID, settle); err != nil {
		t.Fatalf("seed balance: %v", err)
	}
}

func (h *webhookIntegrationHarness) seedIncome(t *testing.T, ctx context.Context, ggr int64, feeTransaction int64, feeWithdrawal int64, amount int64) {
	t.Helper()

	if _, err := h.pool.Exec(ctx, `
		INSERT INTO incomes (ggr, fee_transaction, fee_withdrawal, amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, ggr, feeTransaction, feeWithdrawal, amount); err != nil {
		t.Fatalf("seed income: %v", err)
	}
}

func (h *webhookIntegrationHarness) setTokoCallbackURL(t *testing.T, ctx context.Context, tokoID int64, callbackURL string) {
	t.Helper()

	if _, err := h.pool.Exec(ctx, `
		UPDATE tokos
		SET callback_url = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, tokoID, callbackURL); err != nil {
		t.Fatalf("set toko callback url: %v", err)
	}
}

func (h *webhookIntegrationHarness) seedTransaction(t *testing.T, ctx context.Context, transactionID int64, tokoID int64, category string, transactionType string, status string, amount int64, code string, note string) {
	t.Helper()

	if _, err := h.pool.Exec(ctx, `
		INSERT INTO transactions (id, toko_id, category, type, status, amount, code, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, transactionID, tokoID, category, transactionType, status, amount, code, note); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
}

type integrationTransaction struct {
	Status string
	Player *string
	Note   *string
}

type integrationBalance struct {
	Settle   int64
	Pending  int64
	Nexusggr int64
}

type integrationIncome struct {
	Amount int64
}

type capturingRelayScheduler struct {
	payloads []jobs.TokoCallbackPayload
}

func (c *capturingRelayScheduler) EnqueueRelayTokoCallback(_ context.Context, payload jobs.TokoCallbackPayload) error {
	c.payloads = append(c.payloads, payload)
	return nil
}

func (h *webhookIntegrationHarness) fetchTransaction(t *testing.T, ctx context.Context, code string) integrationTransaction {
	t.Helper()

	var result integrationTransaction
	if err := h.pool.QueryRow(ctx, `
		SELECT status, player, note
		FROM transactions
		WHERE code = $1
		LIMIT 1
	`, code).Scan(&result.Status, &result.Player, &result.Note); err != nil {
		t.Fatalf("fetch transaction: %v", err)
	}
	return result
}

func (h *webhookIntegrationHarness) fetchBalance(t *testing.T, ctx context.Context, tokoID int64) integrationBalance {
	t.Helper()

	var result integrationBalance
	if err := h.pool.QueryRow(ctx, `
		SELECT settle::bigint, pending::bigint, nexusggr::bigint
		FROM balances
		WHERE toko_id = $1
		LIMIT 1
	`, tokoID).Scan(&result.Settle, &result.Pending, &result.Nexusggr); err != nil {
		t.Fatalf("fetch balance: %v", err)
	}
	return result
}

func (h *webhookIntegrationHarness) fetchIncome(t *testing.T, ctx context.Context) integrationIncome {
	t.Helper()

	var result integrationIncome
	if err := h.pool.QueryRow(ctx, `
		SELECT amount::bigint
		FROM incomes
		ORDER BY id
		LIMIT 1
	`).Scan(&result.Amount); err != nil {
		t.Fatalf("fetch income: %v", err)
	}
	return result
}

func decodeJSONMap(t *testing.T, raw *string) map[string]any {
	t.Helper()

	if raw == nil {
		t.Fatal("expected JSON note, got nil")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(*raw), &payload); err != nil {
		t.Fatalf("decode json note: %v", err)
	}

	return payload
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("reserve free port: %w", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

func stringPointer(value string) *string {
	return &value
}
