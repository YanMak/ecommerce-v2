// Layer: Application Port (Outbound)
package repo

import (
	"context"
	"time"

	"github.com/YanMak/ecommerce/v2/services/items/internal/app/usecase/paging"
	"github.com/YanMak/ecommerce/v2/services/items/internal/domain"
)

type ItemPatch struct {
	Name          *string
	Description   *string
	PriceCents    *int64
	Tags          *[]string
	PrevUpdatedAt *time.Time // если делаешь optimistic-версию
}

type SearchFilter struct {
	Name     *string
	MinPrice *int64
	MaxPrice *int64
}

// Layer: Application Port (Outbound)
type ItemRepository interface {
	Create(ctx context.Context, in domain.Item) (domain.Item, error)

	UpsertBySlug(ctx context.Context, in domain.Item) (domain.Item, error)

	Patch(ctx context.Context, id int64, p ItemPatch) (domain.Item, error)

	ByID(ctx context.Context, id int64) (domain.Item, error)

	// новый метод:
	Search(ctx context.Context, name *string, minPrice *int64, maxPrice *int64, limit, offset int32) ([]domain.Item, error)

	CreateAndSearch(
		ctx context.Context, in domain.Item,
		name *string, minPrice *int64, maxPrice *int64, limit, offset int32,
	) ([]domain.Item, error)

	CreateAndSearchWithRetry(
		ctx context.Context, in domain.Item,
		name *string, minPrice *int64, maxPrice *int64, limit, offset int32,
	) ([]domain.Item, error)

	SearchOffset(ctx context.Context, f SearchFilter, p paging.OffsetPage) ([]domain.Item, int64, bool, error)

	SearchKeysetNext(ctx context.Context, f SearchFilter, limit int32, cur *paging.Cursor) ([]domain.Item, *paging.Cursor, bool, error)

	SearchKeysetPrev(ctx context.Context, f SearchFilter, limit int32, cur *paging.Cursor) ([]domain.Item, *paging.Cursor, bool, error)
}
