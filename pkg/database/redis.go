package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tranvux/boilerplate_golang/pkg/config"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"go.uber.org/zap"
)

type RedisClient struct {
	*redis.Client
}

func NewRedisClient(cfg *config.Config, log logger.Logger) (*RedisClient, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Validate connection
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis server at %s: %w", addr, err)
	}

	log.Info("Successfully connected to Redis server",
		zap.String("host", cfg.Redis.Host),
		zap.Int("port", cfg.Redis.Port),
		zap.Int("db", cfg.Redis.DB),
	)

	return &RedisClient{Client: rdb}, nil
}

func (r *RedisClient) Close() error {
	return lclose(r.Client)
}

func lclose(client *redis.Client) error {
	if client != nil {
		return client.Close()
	}
	return nil
}
