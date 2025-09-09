package crmhandlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/query"
	"github.com/YanMak/ecommerce/v2/pkg/httpx/render"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/app/usecase"
)

func Healtz(uc *usecase.CertificatesUC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(time.Now())
	}
}

func Search(uc *usecase.CertificatesUC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		params := usecase.CertsSearchParams{
			Q:           query.StringPtr(q, "q"),
			Inn:         query.StringPtr(q, "inn"),
			CreatedFrom: query.TimePtrRFC3339(q, "created_from"),
			CreatedTo:   query.TimePtrRFC3339(q, "created_to"),
			UpdatedFrom: query.TimePtrRFC3339(q, "updated_from"),
			UpdatedTo:   query.TimePtrRFC3339(q, "updated_to"),
			CategoryID:  query.Int64Ptr(q, "category_id"),
			Opened:      query.BoolPtr(q, "opened"),
			Page:        query.IntDefault(q, "page", 1),
			PerPage:     query.IntDefault(q, "per_page", 10),
		}
		resp, err := uc.Search(r.Context(), params)
		if err != nil {
			render.GRPCError(w, r, err) // следующий шаг — единый рендер ошибок
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func parseTimePtr(v string) *time.Time {
	if v == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return &t
	}
	return nil
}
func parseInt64Ptr(v string) *int64 {
	if v == "" {
		return nil
	}
	if x, err := strconv.ParseInt(v, 10, 64); err == nil {
		return &x
	}
	return nil
}
func parseBoolPtr(v string) *bool {
	if v == "" {
		return nil
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return &b
	}
	return nil
}
func def(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
