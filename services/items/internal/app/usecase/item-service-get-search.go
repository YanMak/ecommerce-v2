package usecase

import (
	"context"

	repoports "github.com/YanMak/ecommerce/v2/services/items/internal/app/ports/repo"
	"github.com/YanMak/ecommerce/v2/services/items/internal/app/usecase/paging"
	"github.com/YanMak/ecommerce/v2/services/items/internal/domain"
)

func (s *ItemService) Get(ctx context.Context, id int64) (domain.Item, error) {
	return s.repo.ByID(ctx, id)
}

func (s *ItemService) Search(ctx context.Context, name *string, minPrice *int64, maxPrice *int64, limit, offset int32) ([]domain.Item, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.Search(ctx, name, minPrice, maxPrice, limit, offset)
}

func (s *ItemService) SearchOffset(ctx context.Context, f repoports.SearchFilter, p paging.OffsetPage) (paging.OffsetResult[domain.Item], error) {

	items, total, hasNext, err := s.repo.SearchOffset(ctx, f, p)
	if err != nil {
		return paging.OffsetResult[domain.Item]{
			Items: nil, Total: 0, Page: p.Page, PerPage: p.PerPage, HasNext: false,
		}, err
	}
	return paging.OffsetResult[domain.Item]{
		Items: items, Total: total, Page: p.Page, PerPage: p.PerPage, HasNext: hasNext,
	}, nil
}

func (s *ItemService) SearchKeysetNext(ctx context.Context, f repoports.SearchFilter, limit int32, cur *paging.Cursor) (paging.KeysetResult[domain.Item], error) {
	// берём limit+1, чтобы понять hasNext
	items, next, hasNext, err := s.repo.SearchKeysetNext(ctx, f, limit, cur)
	if err != nil {
		return paging.KeysetResult[domain.Item]{}, err

	}
	return paging.KeysetResult[domain.Item]{
		Items: items, Cursor: next, HasNext: hasNext,
	}, nil

}

func (s *ItemService) SearchKeysetPrev(ctx context.Context, f repoports.SearchFilter, limit int32, cur *paging.Cursor) (paging.KeysetResult[domain.Item], error) {
	// берём limit+1, чтобы понять hasNext
	items, next, hasNext, err := s.repo.SearchKeysetPrev(ctx, f, limit, cur)
	if err != nil {
		return paging.KeysetResult[domain.Item]{}, err

	}
	return paging.KeysetResult[domain.Item]{
		Items: items, Cursor: next, HasNext: hasNext,
	}, nil

}
