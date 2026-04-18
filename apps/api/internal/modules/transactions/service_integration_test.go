package transactions

import (
	"context"
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
)

var (
	transactionHarnessOnce sync.Once
	transactionHarness     *transactionIntegrationHarness
	transactionHarnessErr  error
)

type transactionIntegrationHarness struct {
	db   *embeddedpostgres.EmbeddedPostgres
	pool *pgxpool.Pool
}

func TestMain(m *testing.M) {
	code := m.Run()

	if transactionHarness != nil {
		transactionHarness.close()
	}

	os.Exit(code)
}

func TestExpireOverduePendingQRISIntegration(t *testing.T) {
	harness := requireTransactionHarness(t)
	harness.reset(t)

	ctx := context.Background()
	harness.seedToko(t, ctx, 1, 1)
	harness.seedTransaction(t, ctx, 1, 1, "qris", "deposit", "pending", 100000, "trx-overdue", "CURRENT_TIMESTAMP - INTERVAL '31 minutes'")
	harness.seedTransaction(t, ctx, 2, 1, "qris", "deposit", "pending", 100000, "trx-fresh", "CURRENT_TIMESTAMP - INTERVAL '5 minutes'")
	harness.seedTransaction(t, ctx, 3, 1, "qris", "withdrawal", "pending", 100000, "wd-overdue", "CURRENT_TIMESTAMP - INTERVAL '31 minutes'")

	service := NewService(harness.pool)
	expiredCount, err := service.ExpireOverduePendingQRIS(ctx)
	if err != nil {
		t.Fatalf("ExpireOverduePendingQRIS: %v", err)
	}

	if expiredCount != 1 {
		t.Fatalf("expected expiredCount 1, got %d", expiredCount)
	}

	if status := harness.fetchStatus(t, ctx, "trx-overdue"); status != "expired" {
		t.Fatalf("expected overdue qris deposit to expire, got %s", status)
	}
	if status := harness.fetchStatus(t, ctx, "trx-fresh"); status != "pending" {
		t.Fatalf("expected fresh qris deposit to remain pending, got %s", status)
	}
	if status := harness.fetchStatus(t, ctx, "wd-overdue"); status != "pending" {
		t.Fatalf("expected overdue withdrawal to remain pending, got %s", status)
	}
}

func requireTransactionHarness(t *testing.T) *transactionIntegrationHarness {
	t.Helper()

	transactionHarnessOnce.Do(func() {
		transactionHarness, transactionHarnessErr = startTransactionHarness()
	})

	if transactionHarnessErr != nil {
		t.Skipf("skip transactions integration harness: %v", transactionHarnessErr)
	}

	return transactionHarness
}

func startTransactionHarness() (*transactionIntegrationHarness, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}

	basePath, err := os.MkdirTemp("", "justqiuv2-transactions-pg-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	config := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(uint32(port)).
		Database("justqiuv2_transactions_test").
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

	if err := applyTransactionSchema(context.Background(), pool); err != nil {
		pool.Close()
		_ = db.Stop()
		return nil, err
	}

	return &transactionIntegrationHarness{db: db, pool: pool}, nil
}

func applyTransactionSchema(ctx context.Context, pool *pgxpool.Pool) error {
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

func (h *transactionIntegrationHarness) reset(t *testing.T) {
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

func (h *transactionIntegrationHarness) close() {
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

func (h *transactionIntegrationHarness) seedToko(t *testing.T, ctx context.Context, userID int64, tokoID int64) {
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
}

func (h *transactionIntegrationHarness) seedTransaction(t *testing.T, ctx context.Context, transactionID int64, tokoID int64, category string, transactionType string, status string, amount int64, code string, createdAtExpr string) {
	t.Helper()

	query := fmt.Sprintf(`
		INSERT INTO transactions (id, toko_id, category, type, status, amount, code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, %s, CURRENT_TIMESTAMP)
	`, createdAtExpr)

	if _, err := h.pool.Exec(ctx, query, transactionID, tokoID, category, transactionType, status, amount, code); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
}

func (h *transactionIntegrationHarness) fetchStatus(t *testing.T, ctx context.Context, code string) string {
	t.Helper()

	var status string
	if err := h.pool.QueryRow(ctx, `
		SELECT status
		FROM transactions
		WHERE code = $1
		LIMIT 1
	`, code).Scan(&status); err != nil {
		t.Fatalf("fetch transaction status: %v", err)
	}

	return status
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("allocate free port: %w", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type")
	}

	return addr.Port, nil
}
