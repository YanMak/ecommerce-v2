package dbgen_test

import (
	"os"
	"testing"

	pgtest "github.com/YanMak/ecommerce/v2/pkg/pgkit/pgtest"
	"github.com/YanMak/ecommerce/v2/services/items/internal/dbgen"
)

//var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestCreateAndGet(t *testing.T) {

	//ctx, tx := beginTx(t)
	t.Setenv("ITEMS_DB_URL", "postgres://postgres:postgres@localhost:5432/items?sslmode=disable")

	pool := pgtest.PoolFromEnv(t, "ITEMS_DB_URL")

	ctx, tx, cleanup := pgtest.BeginTx(t, pool)
	q := dbgen.New(tx)

	created, err := q.CreateItem(ctx, dbgen.CreateItemParams{
		Slug: "t-1", Name: "Test", Description: "from test", PriceCents: 100, Tags: []string{},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := q.GetItemByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("mismatch")
	}

	cleanup()
}
