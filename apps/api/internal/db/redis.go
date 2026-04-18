package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
)

func NewRedisClient(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	return NewRedisClientForDB(ctx, cfg, cfg.DB)
}

func NewRedisClientForDB(ctx context.Context, cfg config.RedisConfig, dbIndex int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address(),
		Password: cfg.Password,
		DB:       dbIndex,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
