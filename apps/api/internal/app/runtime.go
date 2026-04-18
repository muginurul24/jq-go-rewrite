package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/db"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/queue"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/security"
)

type Runtime struct {
	Config         config.Config
	Logger         zerolog.Logger
	DB             *pgxpool.Pool
	Redis          *redis.Client
	CacheRedis     *redis.Client
	Queue          *queue.Client
	SessionManager *scs.SessionManager
}

func New(ctx context.Context) (*Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg)

	pgPool, err := db.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("bootstrap postgres: %w", err)
	}

	redisClient, err := db.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		pgPool.Close()
		return nil, fmt.Errorf("bootstrap redis: %w", err)
	}

	cacheRedisClient, err := db.NewRedisClientForDB(ctx, cfg.Redis, cfg.Redis.CacheDB)
	if err != nil {
		_ = redisClient.Close()
		pgPool.Close()
		return nil, fmt.Errorf("bootstrap cache redis: %w", err)
	}

	sessionManager := security.NewSessionManager(cfg, redisClient)
	queueClient := queue.NewClient(cfg)

	return &Runtime{
		Config:         cfg,
		Logger:         logger,
		DB:             pgPool,
		Redis:          redisClient,
		CacheRedis:     cacheRedisClient,
		Queue:          queueClient,
		SessionManager: sessionManager,
	}, nil
}

func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}

	var closeErrs []error
	if r.Redis != nil {
		closeErrs = append(closeErrs, r.Redis.Close())
	}

	if r.CacheRedis != nil {
		closeErrs = append(closeErrs, r.CacheRedis.Close())
	}

	if r.Queue != nil {
		closeErrs = append(closeErrs, r.Queue.Close())
	}

	if r.DB != nil {
		done := make(chan struct{})
		go func() {
			r.DB.Close()
			close(done)
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
		}
	}

	return errors.Join(closeErrs...)
}

func newLogger(cfg config.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("app", cfg.App.Name).
		Str("env", cfg.App.Env).
		Logger()

	if cfg.App.Debug {
		return logger.Level(zerolog.DebugLevel)
	}

	return logger.Level(zerolog.InfoLevel)
}
