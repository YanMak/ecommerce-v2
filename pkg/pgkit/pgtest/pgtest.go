package pgtest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolFromEnv создает пул из DSN в переменной окружения (например: ITEMS_DB_URL, CRM_DB_URL).
func PoolFromEnv(t *testing.T, dsnEnv string) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv(dsnEnv)
	if dsn == "" {
		t.Fatalf("env %s is empty", dsnEnv)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DSN (%s): %v", dsnEnv, err)
	}
	// адекватные дефолты для тестов:
	cfg.MaxConnIdleTime = 30 * time.Second
	cfg.MaxConnLifetime = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// BeginTx начинает транзакцию и возвращает ctx, tx и cleanup (сделает ROLLBACK).
func BeginTx(t *testing.T, pool *pgxpool.Pool) (context.Context, Tx, func()) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	cleanup := func() { _ = tx.Rollback(ctx) }
	return ctx, tx, cleanup
}

// WithRollback запускает f в транзакции с авто-ROLLBACK.
func WithRollback(t *testing.T, pool *pgxpool.Pool, f func(ctx context.Context, tx Tx)) {
	t.Helper()
	ctx, tx, rollback := BeginTx(t, pool)
	defer rollback()
	f(ctx, tx)
}
