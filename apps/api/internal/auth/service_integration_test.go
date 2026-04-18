package auth

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/security"
)

var (
	authIntegrationHarnessOnce sync.Once
	authIntegrationHarness     *serviceIntegrationHarness
	authIntegrationHarnessErr  error
)

type serviceIntegrationHarness struct {
	db   *embeddedpostgres.EmbeddedPostgres
	pool *pgxpool.Pool
}

func TestMain(m *testing.M) {
	code := m.Run()

	if authIntegrationHarness != nil {
		authIntegrationHarness.close()
	}

	os.Exit(code)
}

func TestAuthenticateCredentialsSupportsUsernameAndEmailIntegration(t *testing.T) {
	harness := requireServiceIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	service := harness.newService()
	harness.seedUser(t, ctx, serviceUserSeed{
		ID:       11,
		Username: "justqiu-dev",
		Name:     "JustQiu Dev",
		Email:    "justqiu-dev@example.test",
		Password: "justqiu-secret",
		Role:     "dev",
		IsActive: true,
	})

	if _, err := service.AuthenticateCredentials(ctx, "justqiu-dev", "justqiu-secret"); err != nil {
		t.Fatalf("authenticate with username: %v", err)
	}

	if _, err := service.AuthenticateCredentials(ctx, "justqiu-dev@example.test", "justqiu-secret"); err != nil {
		t.Fatalf("authenticate with email: %v", err)
	}
}

func TestAuthenticateCredentialsRejectsInactiveUserIntegration(t *testing.T) {
	harness := requireServiceIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	service := harness.newService()
	harness.seedUser(t, ctx, serviceUserSeed{
		ID:       12,
		Username: "inactive-user",
		Name:     "Inactive User",
		Email:    "inactive@example.test",
		Password: "justqiu-secret",
		Role:     "admin",
		IsActive: false,
	})

	if _, err := service.AuthenticateCredentials(ctx, "inactive-user", "justqiu-secret"); !errors.Is(err, ErrInactiveUser) {
		t.Fatalf("expected ErrInactiveUser, got %v", err)
	}
}

func TestConfirmMFASetupStoresEncryptedStateAndVerifyLoginIntegration(t *testing.T) {
	harness := requireServiceIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	service := harness.newService()
	harness.seedUser(t, ctx, serviceUserSeed{
		ID:       13,
		Username: "mfa-user",
		Name:     "MFA User",
		Email:    "mfa-user@example.test",
		Password: "justqiu-secret",
		Role:     "admin",
		IsActive: true,
	})

	setup, err := service.BeginMFASetup(PublicUser{
		ID:       13,
		Username: "mfa-user",
		Name:     "MFA User",
		Email:    "mfa-user@example.test",
		Role:     "admin",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("begin mfa setup: %v", err)
	}

	setupCode, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}

	recoveryCodes := []string{"ABCD-1234", "EFGH-5678"}
	if err := service.ConfirmMFASetup(ctx, 13, setup.Secret, setupCode, recoveryCodes); err != nil {
		t.Fatalf("confirm mfa setup: %v", err)
	}

	storedUser, err := harness.newRepository().FindUserByID(ctx, 13)
	if err != nil {
		t.Fatalf("find stored user: %v", err)
	}

	if storedUser.MFASecret == nil || strings.TrimSpace(*storedUser.MFASecret) == "" {
		t.Fatalf("expected encrypted mfa secret to be stored")
	}
	if storedUser.MFASecret != nil && *storedUser.MFASecret == setup.Secret {
		t.Fatalf("expected stored mfa secret to be encrypted")
	}
	if storedUser.MFARecovery == nil || strings.TrimSpace(*storedUser.MFARecovery) == "" {
		t.Fatalf("expected encrypted recovery codes to be stored")
	}

	decryptedSecret, decryptedRecoveryCodes := harness.decryptMFAState(t, storedUser)
	if decryptedSecret != setup.Secret {
		t.Fatalf("expected decrypted secret %q, got %q", setup.Secret, decryptedSecret)
	}
	if len(decryptedRecoveryCodes) != len(recoveryCodes) {
		t.Fatalf("expected %d recovery codes, got %d", len(recoveryCodes), len(decryptedRecoveryCodes))
	}

	loginCode, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate login totp code: %v", err)
	}

	publicUser, err := service.VerifyLoginMFA(ctx, 13, loginCode)
	if err != nil {
		t.Fatalf("verify login mfa: %v", err)
	}
	if !publicUser.MFAEnabled {
		t.Fatalf("expected public user to report mfa enabled")
	}
}

func TestVerifyLoginMFAConsumesRecoveryCodeOnceIntegration(t *testing.T) {
	harness := requireServiceIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	service := harness.newService()
	harness.seedUser(t, ctx, serviceUserSeed{
		ID:       14,
		Username: "recovery-user",
		Name:     "Recovery User",
		Email:    "recovery@example.test",
		Password: "justqiu-secret",
		Role:     "admin",
		IsActive: true,
	})

	setup, err := service.BeginMFASetup(PublicUser{
		ID:       14,
		Username: "recovery-user",
		Name:     "Recovery User",
		Email:    "recovery@example.test",
		Role:     "admin",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("begin mfa setup: %v", err)
	}

	setupCode, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}

	recoveryCode := "ABCD-1234"
	if err := service.ConfirmMFASetup(ctx, 14, setup.Secret, setupCode, []string{recoveryCode}); err != nil {
		t.Fatalf("confirm mfa setup: %v", err)
	}

	if _, err := service.VerifyLoginMFA(ctx, 14, strings.ToLower(recoveryCode)); err != nil {
		t.Fatalf("verify recovery code: %v", err)
	}

	if _, err := service.VerifyLoginMFA(ctx, 14, recoveryCode); !errors.Is(err, ErrMFAInvalidCode) {
		t.Fatalf("expected recovery code to be single use, got %v", err)
	}

	storedUser, err := harness.newRepository().FindUserByID(ctx, 14)
	if err != nil {
		t.Fatalf("find stored user: %v", err)
	}
	_, remainingCodes := harness.decryptMFAState(t, storedUser)
	if len(remainingCodes) != 0 {
		t.Fatalf("expected recovery codes to be consumed, got %#v", remainingCodes)
	}
}

func TestDisableMFAClearsEncryptedColumnsIntegration(t *testing.T) {
	harness := requireServiceIntegrationHarness(t)
	harness.reset(t)

	ctx := context.Background()
	service := harness.newService()
	harness.seedUser(t, ctx, serviceUserSeed{
		ID:       15,
		Username: "disable-user",
		Name:     "Disable User",
		Email:    "disable@example.test",
		Password: "justqiu-secret",
		Role:     "superadmin",
		IsActive: true,
	})

	setup, err := service.BeginMFASetup(PublicUser{
		ID:       15,
		Username: "disable-user",
		Name:     "Disable User",
		Email:    "disable@example.test",
		Role:     "superadmin",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("begin mfa setup: %v", err)
	}

	setupCode, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}

	if err := service.ConfirmMFASetup(ctx, 15, setup.Secret, setupCode, []string{"ZZZZ-9999"}); err != nil {
		t.Fatalf("confirm mfa setup: %v", err)
	}

	disableCode, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate disable totp code: %v", err)
	}

	if err := service.DisableMFA(ctx, 15, disableCode); err != nil {
		t.Fatalf("disable mfa: %v", err)
	}

	storedUser, err := harness.newRepository().FindUserByID(ctx, 15)
	if err != nil {
		t.Fatalf("find stored user: %v", err)
	}
	if storedUser.MFASecret != nil {
		t.Fatalf("expected mfa secret to be cleared")
	}
	if storedUser.MFARecovery != nil {
		t.Fatalf("expected mfa recovery codes to be cleared")
	}
}

func requireServiceIntegrationHarness(t *testing.T) *serviceIntegrationHarness {
	t.Helper()

	authIntegrationHarnessOnce.Do(func() {
		authIntegrationHarness, authIntegrationHarnessErr = startServiceIntegrationHarness()
	})

	if authIntegrationHarnessErr != nil {
		t.Skipf("skip auth integration harness: %v", authIntegrationHarnessErr)
	}

	return authIntegrationHarness
}

func startServiceIntegrationHarness() (*serviceIntegrationHarness, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}

	basePath, err := os.MkdirTemp("", "justqiuv2-auth-pg-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	config := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(uint32(port)).
		Database("justqiuv2_auth_test").
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

	if err := applyServiceIntegrationSchema(context.Background(), pool); err != nil {
		pool.Close()
		_ = db.Stop()
		return nil, err
	}

	return &serviceIntegrationHarness{
		db:   db,
		pool: pool,
	}, nil
}

func applyServiceIntegrationSchema(ctx context.Context, pool *pgxpool.Pool) error {
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

	if _, err := pool.Exec(ctx, upSection); err != nil {
		return fmt.Errorf("apply integration schema: %w", err)
	}

	return nil
}

type serviceUserSeed struct {
	ID       int64
	Username string
	Name     string
	Email    string
	Password string
	Role     string
	IsActive bool
}

func (h *serviceIntegrationHarness) seedUser(t *testing.T, ctx context.Context, input serviceUserSeed) {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if _, err := h.pool.Exec(ctx, `
		INSERT INTO users (id, username, name, email, password, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, input.ID, input.Username, input.Name, input.Email, string(passwordHash), input.Role, input.IsActive); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func (h *serviceIntegrationHarness) reset(t *testing.T) {
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

func (h *serviceIntegrationHarness) newRepository() *Repository {
	return NewRepository(h.pool)
}

func (h *serviceIntegrationHarness) newService() *Service {
	return NewService(h.newRepository(), security.NewStringCipher("integration-mfa-secret"))
}

func (h *serviceIntegrationHarness) decryptMFAState(t *testing.T, user *User) (string, []string) {
	t.Helper()

	if user.MFASecret == nil || user.MFARecovery == nil {
		t.Fatalf("expected encrypted mfa state to exist")
	}

	cipher := security.NewStringCipher("integration-mfa-secret")
	secret, err := cipher.DecryptString(*user.MFASecret)
	if err != nil {
		t.Fatalf("decrypt mfa secret: %v", err)
	}

	payload, err := cipher.DecryptString(*user.MFARecovery)
	if err != nil {
		t.Fatalf("decrypt recovery codes: %v", err)
	}

	var recoveryCodes []string
	if err := json.Unmarshal([]byte(payload), &recoveryCodes); err != nil {
		t.Fatalf("unmarshal recovery codes: %v", err)
	}

	return secret, recoveryCodes
}

func (h *serviceIntegrationHarness) close() {
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
