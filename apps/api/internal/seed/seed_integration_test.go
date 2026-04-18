package seed

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

func TestDevelopmentSeed(t *testing.T) {
	t.Parallel()

	harness := newSeedHarness(t)
	defer harness.close(t)

	result, err := Development(context.Background(), harness.db)
	if err != nil {
		t.Fatalf("development seed failed: %v", err)
	}

	if result.Username != defaultDevUsername {
		t.Fatalf("unexpected dev username: %s", result.Username)
	}

	var userCount int
	if err := harness.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected 1 user after development seed, got %d", userCount)
	}

	var incomeCount int
	var ggr, feeTx, feeWithdraw int
	if err := harness.db.QueryRow(`
		SELECT COUNT(*), MAX(ggr), MAX(fee_transaction), MAX(fee_withdrawal)
		FROM incomes
	`).Scan(&incomeCount, &ggr, &feeTx, &feeWithdraw); err != nil {
		t.Fatalf("count incomes: %v", err)
	}

	if incomeCount != 1 {
		t.Fatalf("expected 1 income row after development seed, got %d", incomeCount)
	}
	if ggr != defaultIncomeGGR || feeTx != defaultIncomeFeeTransaction || feeWithdraw != defaultIncomeFeeWithdrawal {
		t.Fatalf("unexpected income baseline: ggr=%d feeTx=%d feeWithdraw=%d", ggr, feeTx, feeWithdraw)
	}
}

func TestDemoSeed(t *testing.T) {
	t.Parallel()

	harness := newSeedHarness(t)
	defer harness.close(t)

	result, err := Demo(context.Background(), harness.db)
	if err != nil {
		t.Fatalf("demo seed failed: %v", err)
	}

	if result.TokoToken == "" {
		t.Fatal("expected demo toko token to be populated")
	}

	var userCount, tokoCount, bankCount, playerCount, transactionCount int
	if err := harness.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if err := harness.db.QueryRow(`SELECT COUNT(*) FROM tokos`).Scan(&tokoCount); err != nil {
		t.Fatalf("count tokos: %v", err)
	}
	if err := harness.db.QueryRow(`SELECT COUNT(*) FROM banks`).Scan(&bankCount); err != nil {
		t.Fatalf("count banks: %v", err)
	}
	if err := harness.db.QueryRow(`SELECT COUNT(*) FROM players`).Scan(&playerCount); err != nil {
		t.Fatalf("count players: %v", err)
	}
	if err := harness.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&transactionCount); err != nil {
		t.Fatalf("count transactions: %v", err)
	}

	if userCount != 3 {
		t.Fatalf("expected 3 users after demo seed, got %d", userCount)
	}
	if tokoCount != 1 || bankCount != 1 || playerCount != 1 {
		t.Fatalf("unexpected demo inventory: tokos=%d banks=%d players=%d", tokoCount, bankCount, playerCount)
	}
	if transactionCount != 6 {
		t.Fatalf("expected 6 transactions after demo seed, got %d", transactionCount)
	}

	var incomeAmount int64
	if err := harness.db.QueryRow(`SELECT amount FROM incomes ORDER BY id LIMIT 1`).Scan(&incomeAmount); err != nil {
		t.Fatalf("load income amount: %v", err)
	}
	if incomeAmount != 65000 {
		t.Fatalf("expected seeded income amount 65000, got %d", incomeAmount)
	}

	var tokenableType string
	if err := harness.db.QueryRow(`SELECT tokenable_type FROM personal_access_tokens LIMIT 1`).Scan(&tokenableType); err != nil {
		t.Fatalf("load tokenable_type: %v", err)
	}
	if tokenableType != `App\\Models\\Toko` {
		t.Fatalf("unexpected tokenable_type: %s", tokenableType)
	}
}

type seedHarness struct {
	db         *sql.DB
	postgres   *embeddedpostgres.EmbeddedPostgres
	cleanupDir string
}

func newSeedHarness(t *testing.T) *seedHarness {
	t.Helper()

	port := 17000 + (time.Now().Nanosecond() % 1000)
	basePath, err := os.MkdirTemp("", "justqiuv2-seed-pg-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	config := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(uint32(port)).
		Database("justqiuv2_seed_test").
		Username("postgres").
		Password("postgres").
		BinariesPath(filepath.Join(basePath, "bin")).
		DataPath(filepath.Join(basePath, "data")).
		RuntimePath(filepath.Join(basePath, "run")).
		CachePath(filepath.Join(basePath, "cache")).
		StartTimeout(2 * time.Minute)

	pg := embeddedpostgres.NewDatabase(config)
	if err := pg.Start(); err != nil {
		_ = os.RemoveAll(basePath)
		t.Fatalf("start embedded postgres: %v", err)
	}

	db, err := sql.Open("postgres", config.GetConnectionURL())
	if err == nil {
		_ = db.Close()
		db = nil
	}
	db, err = sql.Open("postgres", fmt.Sprintf("%s?sslmode=disable", config.GetConnectionURL()))
	if err != nil {
		_ = pg.Stop()
		_ = os.RemoveAll(basePath)
		t.Fatalf("open embedded postgres: %v", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		_ = pg.Stop()
		_ = os.RemoveAll(basePath)
		t.Fatalf("ping embedded postgres: %v", err)
	}

	if err := applySeedSchema(context.Background(), db); err != nil {
		_ = db.Close()
		_ = pg.Stop()
		_ = os.RemoveAll(basePath)
		t.Fatalf("apply seed schema: %v", err)
	}

	return &seedHarness{
		db:         db,
		postgres:   pg,
		cleanupDir: basePath,
	}
}

func (h *seedHarness) close(t *testing.T) {
	t.Helper()

	if h.db != nil {
		_ = h.db.Close()
	}
	if h.postgres != nil {
		_ = h.postgres.Stop()
	}
	if h.cleanupDir != "" {
		_ = os.RemoveAll(h.cleanupDir)
	}
}

func applySeedSchema(ctx context.Context, db *sql.DB) error {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("resolve current file path")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", "00001_legacy_core.sql")
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	upSection := string(content)
	if marker := strings.Index(upSection, "-- +goose Down"); marker >= 0 {
		upSection = upSection[:marker]
	}
	upSection = strings.ReplaceAll(upSection, "-- +goose Up", "")

	if _, err := db.ExecContext(ctx, upSection); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}

	return nil
}
