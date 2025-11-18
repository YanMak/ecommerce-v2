package repoports

import (
	"context"

	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/services/telemetry/internal/app/contracts"
)

// Контракт репозитория
type CertificatesRepo interface {
	Search(ctx context.Context, f contracts.SearchFilter, p paging.OffsetParams) (rows []contracts.CertificateRow, total int64, hasNext bool, err error)

	// NEW:
	UpsertDocument(ctx context.Context, d contracts.UpsertDocument) error

	// НОВОЕ:
	UpsertCertificateMin(ctx context.Context, in contracts.UpsertCertificateMin) error
}
