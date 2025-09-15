package redisx

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	cfg "github.com/YanMak/ecommerce/v2/pkg/config"
)

type Client = redis.Client

// NewFromEnv создаёт прод-безопасный клиент Redis с дефолтными таймаутами/пулом.
func NewFromEnv() *Client {
	addr := cfg.Str("REDIS_ADDR", "localhost:6379")
	pass := cfg.Str("REDIS_PASSWORD", "")
	db := cfg.Int("REDIS_DB", 0)

	dialTO := cfg.Dur("REDIS_DIAL_TIMEOUT", 300*time.Millisecond)
	readTO := cfg.Dur("REDIS_READ_TIMEOUT", 500*time.Millisecond)
	writeTO := cfg.Dur("REDIS_WRITE_TIMEOUT", 500*time.Millisecond)
	poolSz := cfg.Int("REDIS_POOL_SIZE", 20) // подправишь под свой RPS

	return redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     pass,
		DB:           db,
		DialTimeout:  dialTO,
		ReadTimeout:  readTO,
		WriteTimeout: writeTO,
		PoolSize:     poolSz,
	})
}

// ReadyCheck — быстрый ping с коротким таймаутом (под /readyz).
func ReadyCheck(ctx context.Context, rdb *Client) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	return rdb.Ping(ctx).Err()
}
