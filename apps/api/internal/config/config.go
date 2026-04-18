package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App           AppConfig
	HTTP          HTTPConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Session       SessionConfig
	Queue         QueueConfig
	Observability ObservabilityConfig
	Integrations  IntegrationsConfig
}

type AppConfig struct {
	Name     string
	Env      string
	Debug    bool
	URL      string
	Locale   string
	Timezone string
}

type HTTPConfig struct {
	Port              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	CacheDB  int
}

type SessionConfig struct {
	Driver             string
	LifetimeMinutes    int
	Domain             string
	SecureCookie       bool
	SameSite           string
	Secret             string
	CSRFSecret         string
	CSRFTrustedOrigins []string
	TokenEncryptionKey string
}

type QueueConfig struct {
	Connection        string
	WorkerConcurrency int
}

type ObservabilityConfig struct {
	PrometheusEnabled bool
	OTLPEndpoint      string
}

type IntegrationsConfig struct {
	QRIS     QRISConfig
	NexusGGR NexusGGRConfig
}

type QRISConfig struct {
	BaseURL    string
	Client     string
	ClientKey  string
	GlobalUUID string
}

type NexusGGRConfig struct {
	BaseURL    string
	AgentCode  string
	AgentToken string
}

func Load() (Config, error) {
	_ = loadDotEnv()

	cfg := Config{
		App: AppConfig{
			Name:     getEnv("APP_NAME", "JustQiuV2 Rewrite"),
			Env:      getEnv("APP_ENV", "development"),
			Debug:    getBool("APP_DEBUG", true),
			URL:      getEnv("APP_URL", "http://localhost:8080"),
			Locale:   getEnv("APP_LOCALE", "en"),
			Timezone: getEnv("APP_TIMEZONE", "Asia/Jakarta"),
		},
		HTTP: HTTPConfig{
			Port:              getEnv("HTTP_PORT", "8080"),
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   15 * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_DATABASE", ""),
			User:     getEnv("DB_USERNAME", ""),
			Password: getEnv("DB_PASSWORD", ""),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
			CacheDB:  getInt("REDIS_CACHE_DB", 1),
		},
		Session: SessionConfig{
			Driver:             getEnv("SESSION_DRIVER", "redis"),
			LifetimeMinutes:    getInt("SESSION_LIFETIME", 120),
			Domain:             getEnv("SESSION_DOMAIN", "localhost"),
			SecureCookie:       getBool("SESSION_SECURE_COOKIE", false),
			SameSite:           strings.ToLower(getEnv("SESSION_SAME_SITE", "lax")),
			Secret:             getEnv("SESSION_SECRET", ""),
			CSRFSecret:         getEnv("CSRF_SECRET", ""),
			CSRFTrustedOrigins: getCSV("CSRF_TRUSTED_ORIGINS"),
			TokenEncryptionKey: getEnv("TOKEN_DISPLAY_ENCRYPTION_KEY", ""),
		},
		Queue: QueueConfig{
			Connection:        getEnv("QUEUE_CONNECTION", "redis"),
			WorkerConcurrency: getInt("QUEUE_WORKER_CONCURRENCY", 10),
		},
		Observability: ObservabilityConfig{
			PrometheusEnabled: getBool("PROMETHEUS_ENABLED", true),
			OTLPEndpoint:      getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		},
		Integrations: IntegrationsConfig{
			QRIS: QRISConfig{
				BaseURL:    getEnv("QRIS_BASE_URL", ""),
				Client:     getEnv("QRIS_CLIENT", ""),
				ClientKey:  getEnv("QRIS_CLIENT_KEY", ""),
				GlobalUUID: getEnv("QRIS_GLOBAL_UUID", ""),
			},
			NexusGGR: NexusGGRConfig{
				BaseURL:    getEnv("NEXUSGGR_BASE_URL", ""),
				AgentCode:  getEnv("NEXUSGGR_AGENT_CODE", ""),
				AgentToken: getEnv("NEXUSGGR_AGENT_TOKEN", ""),
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func loadDotEnv() error {
	path, found := findDotEnvFile()
	if !found {
		return nil
	}

	return godotenv.Load(path)
}

func findDotEnvFile() (string, bool) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	return findDotEnvFileFrom(workingDir)
}

func findDotEnvFileFrom(startDir string) (string, bool) {
	currentDir := startDir

	for {
		candidate := filepath.Join(currentDir, ".env")
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, true
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", false
		}

		currentDir = parentDir
	}
}

func (c Config) Validate() error {
	required := map[string]string{
		"DB_DATABASE":                  c.Database.Name,
		"DB_USERNAME":                  c.Database.User,
		"SESSION_SECRET":               c.Session.Secret,
		"CSRF_SECRET":                  c.Session.CSRFSecret,
		"TOKEN_DISPLAY_ENCRYPTION_KEY": c.Session.TokenEncryptionKey,
	}

	var missing []string
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if len(c.Session.CSRFSecret) < 32 {
		return errors.New("CSRF_SECRET must be at least 32 characters")
	}

	if len(c.Session.TokenEncryptionKey) < 32 {
		return errors.New("TOKEN_DISPLAY_ENCRYPTION_KEY must be at least 32 characters")
	}

	if _, err := time.LoadLocation(c.App.Timezone); err != nil {
		return fmt.Errorf("invalid APP_TIMEZONE %q: %w", c.App.Timezone, err)
	}

	switch c.Session.SameSite {
	case "lax", "strict", "none":
	default:
		return errors.New("SESSION_SAME_SITE must be one of: lax, strict, none")
	}

	return nil
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
	)
}

func (c RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getInt(key string, fallback int) int {
	value, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getBool(key string, fallback bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getCSV(key string) []string {
	value, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}

	if len(values) == 0 {
		return nil
	}

	return values
}
