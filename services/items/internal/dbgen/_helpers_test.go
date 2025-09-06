package dbgen_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func beginTx(t *testing.T) (context.Context, pgx.Tx) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	t.Cleanup(func() { _ = tx.Rollback(ctx) }) // всегда откатываем
	return ctx, tx
}
