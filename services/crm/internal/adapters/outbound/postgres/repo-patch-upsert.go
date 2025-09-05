package postgres

import (
	"context"

	"example.com/sqlchello/internal/core/domain"
	repoports "example.com/sqlchello/internal/core/ports/repo"
	"example.com/sqlchello/internal/dbgen"
	// "context"
	// "example.com/sqlchello/internal/core/domain"
	// ports "example.com/sqlchello/internal/core/ports"
	// "example.com/sqlchello/internal/dbgen"
)

// toUpsert: domain -> dbgen params
func toUpsert(i domain.Item) dbgen.UpsertItemBySlugParams {
	return dbgen.UpsertItemBySlugParams{
		Slug:        i.Slug,
		Name:        i.Name,
		Description: i.Description,
		PriceCents:  i.PriceCents,
		Tags:        i.Tags, // если sqlc сгенерил []string. Если pgtype.TextArray — напишем конвертер, скажи.
	}
}

// Публичный (через pool)
func (r *Repo) UpsertBySlug(ctx context.Context, in domain.Item) (domain.Item, error) {
	// row, err := r.q(r.pool).UpsertItemBySlug(ctx, toUpsert(in))
	// if err != nil { return domain.Item{}, err }
	// return fromDB(row), nil
	return r.UpsertBySlugWith(ctx, r.pool, in)
}

// В транзакции (DBTX)
func (r *Repo) UpsertBySlugWith(ctx context.Context, db dbgen.DBTX, in domain.Item) (domain.Item, error) {
	row, err := r.q(db).UpsertItemBySlug(ctx, toUpsert(in))
	if err != nil {
		return domain.Item{}, err
	}
	return fromDB(row), nil
}

func (r *Repo) Patch(ctx context.Context, id int64, p repoports.ItemPatch) (domain.Item, error) {
	return r.PatchWith(ctx, r.pool, id, p)
}

func (r *Repo) PatchWith(ctx context.Context, db dbgen.DBTX, id int64, p repoports.ItemPatch) (domain.Item, error) {
	params := dbgen.PatchItemOptimisticParams{
		ID:            id,
		Name:          OptText(p.Name),
		Description:   OptText(p.Description),
		Price:         OptInt8(p.PriceCents),
		Tags:          *p.Tags,
		PrevUpdatedAt: OptTime(p.PrevUpdatedAt),
	}
	// we have simple []string array so dont need to convert to pgtype.TextArray
	// // Tags в params заполнить по факту его реального типа:
	// // 1) если pgtype.TextArray:
	// params.Tags = optTextArray(p.Tags)
	// // 2) если []string: if p.Tags != nil { params.Tags = *p.Tags } else оставь zero (или см. сгенерённое поле)

	row, err := r.q(db).PatchItemOptimistic(ctx, params)
	if err != nil {
		return domain.Item{}, err
	}
	return fromDB(row), nil
}
