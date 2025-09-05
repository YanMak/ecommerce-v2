package usecase

import (
	"context"
	"fmt"

	repoports "github.com/YanMak/ecommerce/v2/services/items/internal/app/ports/repo"
	"github.com/YanMak/ecommerce/v2/services/items/internal/domain"
)

func (s *ItemService) UpsertBySlug(ctx context.Context, in domain.Item) (domain.Item, error) {

	// Здесь можно вставлять валидацию/правила домена.
	if in.Name == "" || in.Slug == "" {
		return domain.Item{}, fmt.Errorf("Validaton error: Name and Slug must be non empty, Name = %s, Slug = %s", in.Name, in.Slug)
	}

	return s.repo.UpsertBySlug(ctx, in)
}

func (s *ItemService) Patch(ctx context.Context, id int64, p repoports.ItemPatch) (domain.Item, error) {

	if id <= 0 {
		return domain.Item{}, fmt.Errorf("id required")
	}

	return s.repo.Patch(ctx, id, p)
}
