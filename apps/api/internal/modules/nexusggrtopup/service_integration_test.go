package nexusggrtopup

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

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
)

var (
	topupHarnessOnce sync.Once
	topupHarness     *topupIntegrationHarness
	topupHarnessErr  error
)

type topupIntegrationHarness struct {
	db   *embeddedpostgres.EmbeddedPostgres
	pool *pgxpool.Pool
}

func TestMain(m *testing.M) {
	code := m.Run()

	if topupHarness != nil {
		topupHarness.close()
	}

	os.Exit(code)
}

func TestBootstrapUsesTieredTopupRuleOnFreshDatabaseIntegration(t *testing.T) {
	harness := requireTopupHarness(t)
	harness.reset(t)

	service := NewService(harness.pool, nil)
	result, err := service.Bootstrap(context.Background(), auth.PublicUser{
		ID:       1,
		Role:     "dev",
		IsActive: true,
	}, nil)
	if err != nil {
		t.Fatalf("bootstrap should not fail without incomes row: %v", err)
	}

	if result.TopupRatio != defaultTopupRatio {
		t.Fatalf("expected default topup ratio %d, got %d", defaultTopupRatio, result.TopupRatio)
	}
	if result.TopupRule.ThresholdAmount != discountedTopupThreshold {
		t.Fatalf("expected threshold %d, got %d", discountedTopupThreshold, result.TopupRule.ThresholdAmount)
	}
	if result.TopupRule.BelowThresholdRate != defaultTopupRatio {
		t.Fatalf("expected below threshold rate %d, got %d", defaultTopupRatio, result.TopupRule.BelowThresholdRate)
	}
	if result.TopupRule.AboveThresholdRate != discountedTopupRatio {
		t.Fatalf("expected above threshold rate %d, got %d", discountedTopupRatio, result.TopupRule.AboveThresholdRate)
	}
	if len(result.Tokos) != 0 {
		t.Fatalf("expected no tokos on fresh database, got %d", len(result.Tokos))
	}
}

func requireTopupHarness(t *testing.T) *topupIntegrationHarness {
	t.Helper()

	topupHarnessOnce.Do(func() {
		topupHarness, topupHarnessErr = startTopupHarness()
	})

	if topupHarnessErr != nil {
		t.Skipf("skip topup integration harness: %v", topupHarnessErr)
	}

	return topupHarness
}

func startTopupHarness() (*topupIntegrationHarness, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}

	basePath, err := os.MkdirTemp("", "justqiuv2-topup-pg-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	config := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(uint32(port)).
		Database("justqiuv2_topup_test").
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

	if err := applyTopupSchema(context.Background(), pool); err != nil {
		pool.Close()
		_ = db.Stop()
		return nil, err
	}

	return &topupIntegrationHarness{db: db, pool: pool}, nil
}

func applyTopupSchema(ctx context.Context, pool *pgxpool.Pool) error {
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

func (h *topupIntegrationHarness) reset(t *testing.T) {
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

func (h *topupIntegrationHarness) close() {
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
