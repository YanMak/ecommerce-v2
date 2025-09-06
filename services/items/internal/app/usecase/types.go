package usecase

import "github.com/YanMak/ecommerce/v2/services/items/internal/domain"

type ItemsPage struct {
	Items   []domain.Item
	Total   int64
	Page    int32
	PerPage int32
	HasNext bool
}
