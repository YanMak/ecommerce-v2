package crmhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	cbreaker "github.com/YanMak/ecommerce/v2/pkg/cbreaker"
	"github.com/YanMak/ecommerce/v2/pkg/httpx/bind"
	"github.com/YanMak/ecommerce/v2/services/api-gateway-simple/internal/adapters/inbound/httpapi/dto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func UpsertDocument(
	cli crmpb.CertificatesClient,
	cb *cbreaker.Breaker,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		in, _ := bind.DTO[dto.UpsertDocumentDTO](r) // валидный DTO уже в контексте

		req := &crmpb.UpsertDocumentRequest{
			Id:            in.ID,
			CertificateId: in.CertificateID,
			Url:           in.URL,
			UrlMachine:    in.URLMachine,
			// можем также прокинуть ключ для трейсинга/логов:
			IdempotencyKey: r.Header.Get("Idempotency-Key"),
		}

		var resp *crmpb.UpsertDocumentResponse
		err := cb.Execute(
			r.Context(),
			func(ctx context.Context) error {
				var e error
				resp, e = cli.UpsertDocument(ctx, req)
				return e
			})
		if err != nil {
			if errors.Is(err, cbreaker.ErrCircuitOpen) {
				// быстрый отказ, когда CRM часто падает/тормозит
				if span := trace.SpanFromContext(r.Context()); span != nil {
					span.SetAttributes(attribute.Bool("breaker.rejected", true))
				}
				w.Header().Set("Retry-After", "1")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}
