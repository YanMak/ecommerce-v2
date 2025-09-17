package crmhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	cbreaker "github.com/YanMak/ecommerce/v2/pkg/cbreaker"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type UpsertCertificateMinDTO struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	CategoryID   int64  `json:"categoryId"`
	Opened       bool   `json:"opened"`
	EntityTypeID int64  `json:"entityTypeId"`
	UfUUID       string `json:"ufUuid"`
}

func UpsertCertificateMin(
	cli crmpb.CertificatesClient,
	cb *cbreaker.Breaker,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in UpsertCertificateMinDTO
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}

		req := &crmpb.UpsertCertificateMinRequest{
			Id:           in.ID,
			Title:        in.Title,
			CategoryId:   in.CategoryID,
			Opened:       in.Opened,
			EntityTypeId: in.EntityTypeID,
			UfUuid:       in.UfUUID,
		}

		var resp *crmpb.UpsertCertificateMinResponse
		err := cb.Execute(r.Context(), func(ctx context.Context) error {
			var e error
			resp, e = cli.UpsertCertificateMin(ctx, req)
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
