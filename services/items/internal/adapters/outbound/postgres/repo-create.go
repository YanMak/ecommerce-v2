package postgres

import (
	"context"

	"github.com/YanMak/ecommerce/v2/services/items/internal/dbgen"
	"github.com/YanMak/ecommerce/v2/services/items/internal/domain"
)

// переводчик sqlc -> доменная модель
func fromDB(i dbgen.Item) domain.Item {
	return domain.Item{
		ID: i.ID, Slug: i.Slug, Name: i.Name, Description: i.Description,
		PriceCents: i.PriceCents, Tags: i.Tags, CreatedAt: ToTime(i.CreatedAt), UpdatedAt: ToTime(i.UpdatedAt),
	}
}

// переводчик доменная -> параметры sqlc (для Create)
func toCreate(i domain.Item) dbgen.CreateItemParams {
	return dbgen.CreateItemParams{
		Slug: i.Slug, Name: i.Name, Description: i.Description, PriceCents: i.PriceCents, Tags: i.Tags,
	}
}

func (r *Repo) Create(ctx context.Context, in domain.Item) (domain.Item, error) {
	return r.CreateWith(ctx, r.pool, in)
}

// Реальная работа: вариант "с DBTX"
func (r *Repo) CreateWith(ctx context.Context, db dbgen.DBTX, in domain.Item) (domain.Item, error) {
	row, err := r.q(db).CreateItem(ctx, toCreate(in))
	if err != nil {
		return domain.Item{}, err
	}
	return fromDB(row), nil
}
