package postgres

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	repoports "github.com/YanMak/ecommerce/v2/services/items/internal/app/ports/repo"
	keyset "github.com/YanMak/ecommerce/v2/services/items/internal/app/usecase/paging"
	"github.com/YanMak/ecommerce/v2/services/items/internal/dbgen"
	"github.com/YanMak/ecommerce/v2/services/items/internal/domain"
	"github.com/jackc/pgx/v5"
	// "context"
	// "fmt"
	// "math/rand"
	// "time"
	// "example.com/sqlchello/internal/core/domain"
	// "example.com/sqlchello/internal/core/ports"
	// "example.com/sqlchello/internal/core/usecase/paging"
	// "example.com/sqlchello/internal/dbgen"
	// "github.com/jackc/pgx/v5"
)

func (r *Repo) ByIDWithTX(ctx context.Context, db dbgen.DBTX, id int64) (domain.Item, error) {
	row, err := r.q(r.pool).GetItemByID(context.Background(), id)
	if err != nil {
		return domain.Item{}, err
	}
	return fromDB(row), nil
}

func (r *Repo) ByID(ctx context.Context, id int64) (domain.Item, error) {
	return r.ByIDWithTX(ctx, r.pool, id)
}

func (r *Repo) SearchWith(ctx context.Context, db dbgen.DBTX, name *string, minPrice *int64, maxPrice *int64, limit, offset int32) ([]domain.Item, error) {

	rows, err := r.q(db).SearchItems(
		context.Background(),
		dbgen.SearchItemsParams{
			// имена полей подскажет sqlc в сгенерённом коде
			// чаще всего это Name, MinPrice, Limit, Offset
			// (если имена другие — подсмотри в internal/dbgen/*Search*.go)
			Name:     OptText(name),
			MinPrice: OptInt8(minPrice),
			MaxPrice: OptInt8(maxPrice),
			Limit:    limit,
			Offset:   offset,
		},
	)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromDB(row)) // твой маппинг sqlc → домен
	}
	return out, nil
}

func (r *Repo) Search(ctx context.Context, name *string, minPrice *int64, maxPrice *int64, limit, offset int32) ([]domain.Item, error) {
	return r.SearchWith(ctx, r.pool, name, minPrice, maxPrice, limit, offset)
}

func (r *Repo) CreateAndSearch(
	ctx context.Context, in domain.Item,
	name *string, minPrice *int64, maxPrice *int64, limit, offset int32,
) ([]domain.Item, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return []domain.Item{}, err
	}
	defer tx.Rollback(ctx)

	_, err = r.CreateWith(ctx, tx, in)
	items, err := r.SearchWith(ctx, tx, name, minPrice, maxPrice, limit, offset)

	if err := tx.Commit(ctx); err != nil {
		return []domain.Item{}, err
	}

	return items, nil
}

func (r *Repo) CreateAndSearchWithRetry(
	ctx context.Context, in domain.Item,
	name *string, minPrice *int64, maxPrice *int64, limit, offset int32,
) ([]domain.Item, error) {

	var created dbgen.Item
	rng := rand.New(rand.NewSource(1))

	err := r.InTxRetry(ctx,
		//r.pool,
		pgx.TxOptions{IsoLevel: pgx.Serializable}, // строгая изоляция
		// TxExtra{
		// 	Retries:     3,
		// 	BaseBackoff: 50 * time.Millisecond,
		// 	MaxBackoff:  time.Second,
		// 	StmtTimeout: 1500 * time.Millisecond,
		// 	LockTimeout: 800 * time.Millisecond,
		// },
		TxExtra{
			Retries:     3,
			BaseBackoff: 50 * time.Millisecond,
			MaxBackoff:  500 * time.Second,
			StmtTimeout: 5000 * 1500 * time.Millisecond,
			LockTimeout: 5000 * 800 * time.Millisecond,
		},
		rng,
		func(ctx context.Context, q *dbgen.Queries) error {
			var err error
			created, err = q.CreateItem(ctx, toCreate(in))
			if err != nil {
				return err
			}
			return nil
		},
	)

	fmt.Printf("crarted: +%v", created)
	if err != nil {
		return []domain.Item{}, err
	}

	return []domain.Item{}, nil
}

func (r *Repo) SearchOffset(ctx context.Context, f repoports.SearchFilter, limit, offset int32) ([]domain.Item, int64, bool, error) {
	// 1) данные
	rows, err := r.q(r.pool).SearchItems(ctx, dbgen.SearchItemsParams{
		Name:     OptText(f.Name),
		MinPrice: OptInt8(f.MinPrice),
		MaxPrice: OptInt8(f.MaxPrice),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return []domain.Item{}, 0, false, err
	}

	// 2) total
	total, err := r.q(r.pool).CountItems(ctx, dbgen.CountItemsParams{
		Name:     OptText(f.Name),
		MinPrice: OptInt8(f.MinPrice),
		MaxPrice: OptInt8(f.MaxPrice),
	})
	if err != nil {
		return []domain.Item{}, 0, false, err
	}

	items := make([]domain.Item, len(rows))
	for i, row := range rows {
		items[i] = fromDB(row)
	}

	hasNext := int64(offset)+int64(limit) < total

	return items, total, hasNext, nil
}

func (r *Repo) SearchKeysetNext(ctx context.Context, f repoports.SearchFilter, limit int32, cur *keyset.Cursor) ([]domain.Item, *keyset.Cursor, bool, error) {
	// берём limit+1, чтобы понять hasNext
	lim := limit + 1

	params := dbgen.SearchItemsKeysetNextParams{
		Name:           OptText(f.Name),
		MinPrice:       OptInt8(f.MinPrice),
		MaxPrice:       OptInt8(f.MaxPrice),
		Limit:          lim,
		AfterCreatedAt: OptTime(nil),
		AfterID:        OptInt8(nil),
	}
	if cur != nil {
		params.AfterCreatedAt = OptTime(&cur.CreatedAt)
		params.AfterID = OptInt8(&cur.ID)
	}

	rows, err := r.q(r.pool).SearchItemsKeysetNext(ctx, params)
	if err != nil {
		return []domain.Item{}, nil, false, err
	}

	hasNext := int32(len(rows)) > limit
	if hasNext {
		rows = rows[:limit]
	}

	items := make([]domain.Item, len(rows))
	for i, row := range rows {
		items[i] = fromDB(row)
	}

	// cursor на следующую страницу — это последняя запись в этой выдаче
	var next *keyset.Cursor
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		next = &keyset.Cursor{CreatedAt: last.CreatedAt.Time, ID: last.ID}
	}

	return items, next, hasNext, nil
}

func (r *Repo) SearchKeysetPrev(ctx context.Context, f repoports.SearchFilter, limit int32, cur *keyset.Cursor) ([]domain.Item, *keyset.Cursor, bool, error) {
	// берём limit+1, чтобы понять hasNext
	lim := limit + 1

	params := dbgen.SearchItemsKeysetPrevParams{
		Name:            OptText(f.Name),
		MinPrice:        OptInt8(f.MinPrice),
		MaxPrice:        OptInt8(f.MaxPrice),
		Limit:           lim,
		BeforeCreatedAt: OptTime(nil),
		BeforeID:        OptInt8(nil),
	}
	if cur != nil {
		params.BeforeCreatedAt = OptTime(&cur.CreatedAt)
		params.BeforeID = OptInt8(&cur.ID)
	}

	rows, err := r.q(r.pool).SearchItemsKeysetPrev(ctx, params)
	if err != nil {
		return []domain.Item{}, nil, false, err
	}

	hasNext := int32(len(rows)) > limit
	if hasNext {
		rows = rows[:limit]
	}

	items := make([]domain.Item, len(rows))
	for i, row := range rows {
		items[i] = fromDB(row)
	}

	// cursor на следующую страницу — это последняя запись в этой выдаче
	var next *keyset.Cursor
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		next = &keyset.Cursor{CreatedAt: last.CreatedAt.Time, ID: last.ID}
	}

	return items, next, hasNext, nil
}
