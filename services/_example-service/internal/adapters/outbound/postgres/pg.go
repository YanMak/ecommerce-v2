package postgres

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool = pgxpool.Pool

// NewPool — создаёт пул подключений к Postgres.
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 4
	cfg.MinConns = 0
	cfg.HealthCheckPeriod = 30 * time.Second
	return pgxpool.NewWithConfig(ctx, cfg)
}
