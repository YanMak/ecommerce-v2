package dbgen_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/YanMak/ecommerce/v2/services/crm/internal/dbgen"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Second)
	defer cancel()

	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable"
	}

	var err error
	pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func TestCreateAndGet(t *testing.T) {
	ctx, tx := beginTx(t)
	q := dbgen.New(tx)

	created, err := q.UpsertCertificate(ctx, dbgen.UpsertCertificateParams{
		ID: 1,
		XmlID: 1,
		Title: "test 1",
		CreatedBy: 0,
		UpdatedBy: 0,
		MovedBy: 0,
		CreatedTime: time.Now(),

	})
	
	
	(ctx, dbgen.CreateItemParams{
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
}
