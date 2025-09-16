package crmhandlers

import (
	"encoding/json"
	"net/http"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	"github.com/YanMak/ecommerce/v2/pkg/httpx/bind"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/adapters/inbound/httpapi/dto"
)

func UpsertDocument(cli crmpb.CertificatesClient) http.HandlerFunc {
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
		resp, err := cli.UpsertDocument(r.Context(), req)
		if err != nil {
			// у тебя уже есть универсальный рендер grpc ошибок; для краткости:
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}
