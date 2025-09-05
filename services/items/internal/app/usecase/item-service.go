// Layer: Application (Use Case)
package usecase

import (
	repoports "github.com/YanMak/ecommerce/v2/services/items/internal/app/ports/repo"
)

type ItemService struct {
	repo repoports.ItemRepository
}

func NewItemService(r repoports.ItemRepository) *ItemService { return &ItemService{repo: r} }
