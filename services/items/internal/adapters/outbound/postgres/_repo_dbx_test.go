package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"example.com/sqlchello/internal/adapters/outbound/postgres"
	"example.com/sqlchello/internal/core/domain"
	"example.com/sqlchello/internal/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

// Общий setup для пакета
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Second)
	defer cancel()

	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/items?sslmode=disable"
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

// Хелперы: tx на тест + откат
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

func TestRepo_CreateWith_Tx(t *testing.T) {
	ctx, tx := beginTx(t)

	// Репозиторий «как в проде»
	repo := postgres.New(pool)

	in := domain.Item{
		Slug:        fmt.Sprintf("t-%d", time.Now().UnixNano()),
		Name:        "Test",
		Description: "repo dbtx",
		PriceCents:  777,
		Tags:        []string{"dbtx"},
	}

	// КЛЮЧ: даём репо не pool, а транзакцию (DBTX)
	created, err := repo.CreateWith(ctx, tx, in)
	if err != nil {
		t.Fatalf("CreateWith: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("want ID > 0")
	}
	if created.Name != in.Name || created.PriceCents != in.PriceCents {
		t.Fatalf("mismatch: got=%+v", created)
	}

	// Доп.проверка: читаем внутри той же tx через sqlc (DBTX-паттерн!)
	qtx := dbgen.New(tx)
	got, err := qtx.GetItemByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetItemByID in tx: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("not found inside tx")
	}

	// Коммит НЕ делаем — t.Cleanup откатит. БД останется чистой.
}
