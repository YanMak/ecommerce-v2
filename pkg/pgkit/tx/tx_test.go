package tx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	pgtest "github.com/YanMak/ecommerce/v2/pkg/pgkit/pgtest"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/tx"
)

func Test_InTx_Commit(t *testing.T) {
	t.Setenv("CRM_DB_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable")
	pool := pgtest.PoolFromEnv(t, "CRM_DB_URL")

	if err := tx.InTx(context.Background(), pool, func(ctx context.Context, _ pgx.Tx) error {
		// no-op
		return nil
	}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func Test_InTx_RollbackOnError(t *testing.T) {
	t.Setenv("CRM_DB_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable")
	pool := pgtest.PoolFromEnv(t, "CRM_DB_URL")

	sentinel := errors.New("boom")
	err := tx.InTx(context.Background(), pool, func(ctx context.Context, _ pgx.Tx) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("want %v, got %v", sentinel, err)
	}
}

func Test_InTx_RollbackOnPanic(t *testing.T) {
	t.Setenv("CRM_DB_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable")
	pool := pgtest.PoolFromEnv(t, "CRM_DB_URL")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic to be rethrown")
		}
	}()

	_ = tx.InTx(context.Background(), pool, func(ctx context.Context, _ pgx.Tx) error {
		panic("panic inside tx")
	})
}
