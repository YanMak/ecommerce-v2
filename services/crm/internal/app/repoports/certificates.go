package repoports

import (
	"context"

	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/contracts"
)

// Контракт репозитория
type CertificatesRepo interface {
	Search(ctx context.Context, f contracts.SearchFilter, p paging.OffsetParams) (rows []contracts.CertificateRow, total int64, hasNext bool, err error)
}
