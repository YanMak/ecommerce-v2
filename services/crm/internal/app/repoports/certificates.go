package repoports

import (
	"context"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/paging"
)

// Входные фильтры (чистые Go-типы)
type SearchFilter struct {
	Q           *string
	Inn         *string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	CategoryID  *int64
	Opened      *bool
}

// Read-модель строки (то, что нужно usecase/handler; без pgtype)
type CertificateRow struct {
	ID          int64
	Title       string
	UfNumber    *string
	UfInn       *string
	CategoryID  int64
	Opened      bool
	CreatedTime *time.Time
	UpdatedTime *time.Time
}

// Контракт репозитория
type CertificatesRepo interface {
	Search(ctx context.Context, f SearchFilter, p paging.OffsetParams) (rows []CertificateRow, total int64, hasNext bool, err error)
}
