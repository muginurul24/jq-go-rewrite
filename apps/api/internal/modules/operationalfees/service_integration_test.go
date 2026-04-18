package operationalfees

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
	"github.com/rs/zerolog"
)

var (
	operationalFeeHarnessOnce sync.Once
	operationalFeeHarness     *operationalFeeIntegrationHarness
	operationalFeeHarnessErr  error
)

type operationalFeeIntegrationHarness struct {
	db   *embeddedpostgres.EmbeddedPostgres
	pool *pgxpool.Pool
}

func TestMain(m *testing.M) {
	code := m.Run()

	if operationalFeeHarness != nil {
		operationalFeeHarness.close()
	}

	os.Exit(code)
}

func TestCollectMonthlyOperationalFeesIntegration(t *testing.T) {
	harness := requireOperationalFeeHarness(t)
	harness.reset(t)

	ctx := context.Background()
	harness.seedIncome(t, ctx, 1_000)
	harness.seedBalance(t, ctx, 1, 150_000)
	harness.seedBalance(t, ctx, 2, 100_000)
	harness.seedBalance(t, ctx, 3, 50_000)
	harness.seedBalance(t, ctx, 4, 0)

	service := NewService(harness.pool, zerolog.Nop())
	result, err := service.CollectMonthlyOperationalFees(ctx)
	if err != nil {
		t.Fatalf("CollectMonthlyOperationalFees: %v", err)
	}

	if result.ProcessedCount != 3 {
		t.Fatalf("expected processed count 3, got %d", result.ProcessedCount)
	}
	if result.DeductedTotal != 250_000 {
		t.Fatalf("expected deducted total 250000, got %d", result.DeductedTotal)
	}

	if settle := harness.fetchSettle(t, ctx, 1); settle != 50_000 {
		t.Fatalf("expected balance 1 settle 50000, got %d", settle)
	}
	if settle := harness.fetchSettle(t, ctx, 2); settle != 0 {
		t.Fatalf("expected balance 2 settle 0, got %d", settle)
	}
	if settle := harness.fetchSettle(t, ctx, 3); settle != 0 {
		t.Fatalf("expected balance 3 settle 0, got %d", settle)
	}
	if settle := harness.fetchSettle(t, ctx, 4); settle != 0 {
		t.Fatalf("expected balance 4 settle 0, got %d", settle)
	}

	if incomeAmount := harness.fetchIncomeAmount(t, ctx); incomeAmount != 251_000 {
		t.Fatalf("expected income amount 251000, got %d", incomeAmount)
	}
}

func requireOperationalFeeHarness(t *testing.T) *operationalFeeIntegrationHarness {
	t.Helper()

	operationalFeeHarnessOnce.Do(func() {
		operationalFeeHarness, operationalFeeHarnessErr = startOperationalFeeHarness()
	})

	if operationalFeeHarnessErr != nil {
		t.Skipf("skip operational fees integration harness: %v", operationalFeeHarnessErr)
	}

	return operationalFeeHarness
}

func startOperationalFeeHarness() (*operationalFeeIntegrationHarness, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}

	basePath, err := os.MkdirTemp("", "justqiuv2-operational-fees-pg-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	config := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(uint32(port)).
		Database("justqiuv2_operational_fees_test").
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

	if err := applyOperationalFeeSchema(context.Background(), pool); err != nil {
		pool.Close()
		_ = db.Stop()
		return nil, err
	}

	return &operationalFeeIntegrationHarness{db: db, pool: pool}, nil
}

func applyOperationalFeeSchema(ctx context.Context, pool *pgxpool.Pool) error {
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

func (h *operationalFeeIntegrationHarness) reset(t *testing.T) {
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

func (h *operationalFeeIntegrationHarness) close() {
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

func (h *operationalFeeIntegrationHarness) seedIncome(t *testing.T, ctx context.Context, amount int64) {
	t.Helper()

	if _, err := h.pool.Exec(ctx, `
		INSERT INTO incomes (ggr, fee_transaction, fee_withdrawal, amount, created_at, updated_at)
		VALUES (7, 3, 15, $1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, amount); err != nil {
		t.Fatalf("seed income: %v", err)
	}
}

func (h *operationalFeeIntegrationHarness) seedBalance(t *testing.T, ctx context.Context, tokoID int64, settle int64) {
	t.Helper()

	userID := tokoID
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

func (h *operationalFeeIntegrationHarness) fetchSettle(t *testing.T, ctx context.Context, tokoID int64) int64 {
	t.Helper()

	var settle int64
	if err := h.pool.QueryRow(ctx, `
		SELECT COALESCE(settle, 0)::bigint
		FROM balances
		WHERE toko_id = $1
	`, tokoID).Scan(&settle); err != nil {
		t.Fatalf("fetch settle: %v", err)
	}

	return settle
}

func (h *operationalFeeIntegrationHarness) fetchIncomeAmount(t *testing.T, ctx context.Context) int64 {
	t.Helper()

	var amount int64
	if err := h.pool.QueryRow(ctx, `
		SELECT amount::bigint
		FROM incomes
		ORDER BY id
		LIMIT 1
	`).Scan(&amount); err != nil {
		t.Fatalf("fetch income amount: %v", err)
	}

	return amount
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("reserve port: %w", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}
