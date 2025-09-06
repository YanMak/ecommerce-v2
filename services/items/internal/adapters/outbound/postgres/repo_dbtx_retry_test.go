package postgres_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/YanMak/ecommerce/v2/services/items/internal/adapters/outbound/postgres"
	repoports "github.com/YanMak/ecommerce/v2/services/items/internal/app/ports/repo"
	"github.com/YanMak/ecommerce/v2/services/items/internal/domain"

	ptr "github.com/YanMak/ecommerce/v2/pkg/ptr"
	"github.com/YanMak/ecommerce/v2/services/items/internal/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

func beginTx(t *testing.T) (context.Context, pgx.Tx) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Second)
	t.Cleanup(cancel)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	return ctx, tx
}

// удерживаем row-lock на items.id = id в отдельной транзакции
func holdRowLock(t *testing.T, id int64, holdFor time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("blocker begin: %v", err)
	}

	qtx := dbgen.New(tx)
	// SELECT ... FOR UPDATE держит блокировку строки до COMMIT/RB
	if _, err := qtx.GetItemByIDForUpdate(ctx, id); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("blocker for update: %v", err)
	}

	// отпустим lock позже
	go func() {
		time.Sleep(holdFor)
		_ = tx.Commit(ctx)
	}()
}

// небольшой helper: создать item (в отдельной транзакции)
func mustCreate(t *testing.T, name, slug string, price int64) (int64, string, time.Time) {
	t.Helper()
	ctx, tx := beginTx(t)
	qtx := dbgen.New(tx)
	row, err := qtx.CreateItem(ctx, dbgen.CreateItemParams{
		Slug: slug, Name: name, Description: "t", PriceCents: price, Tags: []string{},
	})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	// patch before and after commit
	updated1, err := qtx.PatchItemOptimistic(ctx, dbgen.PatchItemOptimisticParams{
		ID:            row.ID,
		Description:   postgres.OptText(ptr.To("changed description 1")),
		PrevUpdatedAt: row.UpdatedAt,
	})
	fmt.Printf("updated in same tx = %+v \n", updated1)

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("seed commit: %v", err)
	}

	// patch after commit
	qtx2 := dbgen.New(pool)
	updated2, err := qtx2.PatchItemOptimistic(ctx, dbgen.PatchItemOptimisticParams{
		ID:            row.ID,
		Description:   postgres.OptText(ptr.To("changed description 2")),
		PrevUpdatedAt: updated1.UpdatedAt,
	})
	fmt.Printf("updated after tx = %+v \n", updated2)

	//return row.ID, postgres.ToTime(row.UpdatedAt)
	return row.ID, updated2.Slug, postgres.ToTime(updated2.UpdatedAt)
}

func TestRepo_Patch_WithRetry_LockTimeout(t *testing.T) {
	repo := postgres.New(pool)

	// 1) готовим строку и держим её под row-lock 300ms
	id, _, updated_at := mustCreate(t, "Seed", fmt.Sprintf("seed-%d", time.Now().UnixNano()), 200)
	holdRowLock(t, id, 300*time.Millisecond)

	// 2) ретрай: первая попытка упадёт по lock_timeout, вторая — пройдёт
	ctx, cancel := context.WithTimeout(context.Background(), 3000*time.Second)
	defer cancel()

	rng := rand.New(rand.NewSource(1))
	attempts := 0

	var priceCents int64 = 2699
	portToRepo := repoports.ItemPatch{
		Name:          ptr.To("saladdin"),
		Description:   ptr.To("new desc while locked"),
		PriceCents:    ptr.To(priceCents),
		Tags:          ptr.To([]string{"sqlc", "hex", "added tag"}),
		PrevUpdatedAt: ptr.To(updated_at),
	}

	var updatedItem dbgen.Item

	err := repo.InTxRetry(
		ctx,
		//pool,
		pgx.TxOptions{IsoLevel: pgx.ReadCommitted},
		postgres.TxExtra{
			Retries:     2,
			BaseBackoff: 50 * time.Millisecond,
			MaxBackoff:  300 * time.Millisecond,
			StmtTimeout: 250 * time.Millisecond,
			LockTimeout: 200 * time.Millisecond,
		},
		rng,
		func(ctx context.Context, q *dbgen.Queries) error {
			attempts++
			var err error
			updatedItem, err = q.PatchItemOptimistic(ctx, dbgen.PatchItemOptimisticParams{
				ID:            id,                                       // ВАЖНО: тот же id, что под lock’ом
				Name:          postgres.OptText(portToRepo.Name),        // не трогаем
				Description:   postgres.OptText(portToRepo.Description), // не трогаем
				Price:         postgres.OptInt8(portToRepo.PriceCents),  // только цену
				Tags:          *portToRepo.Tags,                         // теги не трогаем
				PrevUpdatedAt: postgres.OptTime(portToRepo.PrevUpdatedAt),
			})
			return err
		},
	)
	if err != nil {
		t.Fatalf("InTxRetry: %v", err)
	}
	if attempts < 2 {
		t.Fatalf("ожидали минимум 2 попытки, got %d", attempts)
	}
	fmt.Printf("updated item: +%v", updatedItem)
}

func TestRepo_UpsertBySlug_WithRetry_LockTimeout(t *testing.T) {
	repo := postgres.New(pool)

	// 1) готовим строку и держим её под row-lock 300ms
	id, slug, _ := mustCreate(t, "Seed", fmt.Sprintf("seed-%d", time.Now().UnixNano()), 200)
	holdRowLock(t, id, 300*time.Millisecond)

	// 2) ретрай: первая попытка упадёт по lock_timeout, вторая — пройдёт
	ctx, cancel := context.WithTimeout(context.Background(), 3000*time.Second)
	defer cancel()

	rng := rand.New(rand.NewSource(1))
	attempts := 0

	var priceCents int64 = 3699
	in := domain.Item{
		Slug:        slug,
		Name:        "saladdin",
		Description: "new desc while locked",
		PriceCents:  priceCents,
		Tags:        []string{"sqlc", "hex", "added tag while locked"},
	}

	var upsertedItem dbgen.Item

	err := repo.InTxRetry(
		ctx,
		//pool,
		pgx.TxOptions{IsoLevel: pgx.ReadCommitted},
		postgres.TxExtra{
			Retries:     2,
			BaseBackoff: 50 * time.Millisecond,
			MaxBackoff:  300 * time.Millisecond,
			StmtTimeout: 250 * time.Millisecond,
			LockTimeout: 200 * time.Millisecond,
		},
		rng,
		func(ctx context.Context, q *dbgen.Queries) error {
			attempts++
			var err error
			upsertedItem, err = q.UpsertItemBySlug(ctx, dbgen.UpsertItemBySlugParams{
				Slug:        in.Slug,
				Name:        in.Name,
				Description: in.Description,
				PriceCents:  in.PriceCents,
				Tags:        in.Tags,
			})
			return err
		},
	)
	if err != nil {
		t.Fatalf("InTxRetry: %v", err)
	}
	if attempts < 2 {
		t.Fatalf("ожидали минимум 2 попытки, got %d", attempts)
	}
	fmt.Printf("upsertedItem : +%v", upsertedItem)

	holdRowLock(t, id, 600*time.Millisecond)
	time.Sleep(0 * time.Millisecond)
	newAttempts := 0
	err = repo.InTxRetry(
		ctx,
		//pool,
		pgx.TxOptions{IsoLevel: pgx.ReadCommitted},
		postgres.TxExtra{
			Retries:     3,
			BaseBackoff: 50 * time.Millisecond,
			MaxBackoff:  300 * time.Millisecond,
			StmtTimeout: 250 * time.Millisecond,
			LockTimeout: 200 * time.Millisecond,
		},
		rng,
		func(ctx context.Context, q *dbgen.Queries) error {
			newAttempts++
			var err error
			upsertedItem, err = q.UpsertItemBySlug(ctx, dbgen.UpsertItemBySlugParams{
				Slug:        in.Slug,
				Name:        in.Name,
				Description: in.Description,
				PriceCents:  in.PriceCents,
				Tags:        in.Tags,
			})
			return err
		},
	)
	if err != nil {
		t.Fatalf("InTxRetry: %v", err)
	}
	if newAttempts < 3 {
		t.Fatalf("ожидали минимум 3 попытки, got %d", newAttempts)
	}
	fmt.Printf("upsertedItem : +%v", upsertedItem)

}
