package dbgen_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func nullText() pgtype.Text         { return pgtype.Text{Valid: false} }
func someText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }
func nullInt8() pgtype.Int8         { return pgtype.Int8{Valid: false} }
func someInt8(n int64) pgtype.Int8  { return pgtype.Int8{Int64: n, Valid: true} }
