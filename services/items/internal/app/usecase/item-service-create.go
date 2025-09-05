package usecase

import (
	"context"
	"fmt"

	"github.com/YanMak/ecommerce/v2/services/items/internal/domain"
)

func (s *ItemService) Create(ctx context.Context, in domain.Item) (domain.Item, error) {
	// Здесь можно вставлять валидацию/правила домена.
	if in.Name == "" || in.Slug == "" {
		return domain.Item{}, fmt.Errorf("Validaton error: Name and Slug must be non empty, Name = %s, Slug = %s", in.Name, in.Slug)
	}

	return s.repo.Create(ctx, in)
}
