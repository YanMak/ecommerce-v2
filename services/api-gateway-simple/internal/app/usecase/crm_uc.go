package usecase

import (
	"context"
	"time"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	"github.com/YanMak/ecommerce/v2/pkg/grpcx"
)

// Входные параметры для HTTP-слоя (просто чистые Go-типы)
type CertsSearchParams struct {
	Q, Inn                 *string
	CreatedFrom, CreatedTo *time.Time
	UpdatedFrom, UpdatedTo *time.Time
	CategoryID             *int64
	Opened                 *bool
	Page, PerPage          int
}

// Порт: можно оставить прямо crmpb.CertificatesClient, ок для старта
type CertificatesAPI = crmpb.CertificatesClient

type CertificatesUC struct {
	crm CertificatesAPI
}

func NewCertificatesUC(crm CertificatesAPI) *CertificatesUC {
	return &CertificatesUC{crm: crm}
}

// Возвращаем protobuf-ответ — этого достаточно для шага 1
func (uc *CertificatesUC) Search(ctx context.Context, p CertsSearchParams) (*crmpb.SearchCertificatesResponse, error) {
	req := &crmpb.SearchCertificatesRequest{
		Q:           grpcx.S(p.Q),
		Inn:         grpcx.S(p.Inn),
		CreatedFrom: grpcx.TS(p.CreatedFrom),
		CreatedTo:   grpcx.TS(p.CreatedTo),
		UpdatedFrom: grpcx.TS(p.UpdatedFrom),
		UpdatedTo:   grpcx.TS(p.UpdatedTo),
		CategoryId:  grpcx.I64(p.CategoryID),
		Opened:      grpcx.B(p.Opened),
		Page:        int32(p.Page),
		PerPage:     int32(p.PerPage),
	}
	// client-interceptor сам положит request-id/idempotency-key из ctx в metadata
	return uc.crm.SearchCertificates(ctx, req)
}
