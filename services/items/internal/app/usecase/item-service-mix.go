package usecase

import (
	"context"
	"fmt"

	"github.com/YanMak/ecommerce/v2/services/items/internal/domain"
)

func (s *ItemService) CreateAndSearch(ctx context.Context, in domain.Item,
	name *string, minPrice *int64, maxPrice *int64, limit, offset int32) ([]domain.Item, error) {
	// Здесь можно вставлять валидацию/правила домена.
	if in.Name == "" || in.Slug == "" {
		return []domain.Item{}, fmt.Errorf("Validaton error: Name and Slug must be non empty, Name = %s, Slug = %s", in.Name, in.Slug)
	}

	return s.repo.CreateAndSearch(ctx, in,
		name, minPrice, maxPrice, limit, offset)
}

func (s *ItemService) CreateAndSearchWithTx(ctx context.Context, in domain.Item,
	name *string, minPrice *int64, maxPrice *int64, limit, offset int32) ([]domain.Item, error) {
	// Здесь можно вставлять валидацию/правила домена.
	if in.Name == "" || in.Slug == "" {
		return []domain.Item{}, fmt.Errorf("Validaton error: Name and Slug must be non empty, Name = %s, Slug = %s", in.Name, in.Slug)
	}

	return s.repo.CreateAndSearch(ctx, in,
		name, minPrice, maxPrice, limit, offset)
}

func (s *ItemService) CreateAndSearchWithRetry(ctx context.Context, in domain.Item,
	name *string, minPrice *int64, maxPrice *int64, limit, offset int32) ([]domain.Item, error) {
	// Здесь можно вставлять валидацию/правила домена.
	if in.Name == "" || in.Slug == "" {
		return []domain.Item{}, fmt.Errorf("Validaton error: Name and Slug must be non empty, Name = %s, Slug = %s", in.Name, in.Slug)
	}

	return s.repo.CreateAndSearchWithRetry(ctx, in,
		name, minPrice, maxPrice, limit, offset)
}
