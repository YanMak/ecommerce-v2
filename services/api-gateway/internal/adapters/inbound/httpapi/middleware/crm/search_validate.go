package crmapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/render"
)

// Параметры можно вынести в конфиг при желании
const (
	maxPerPage = 200
)

// ValidateCertsSearch — пред-валидация запроса /crm/certificates/search.
// Проверяем page/per_page и диапазоны дат. Если есть проблемы — сразу 400 JSON.
func ValidateCertsSearch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var v []render.Violation

		// page (если задан) — целое >=1
		if s := q.Get("page"); s != "" {
			if x, ok := parseInt(s); !ok || x < 1 {
				v = append(v, render.Violation{Field: "page", Reason: "must be >= 1"})
			}
		}
		// per_page (если задан) — целое 1..maxPerPage
		if s := q.Get("per_page"); s != "" {
			if x, ok := parseInt(s); !ok || x < 1 || x > maxPerPage {
				v = append(v, render.Violation{Field: "per_page", Reason: "must be in [1..200]"})
			}
		}
		// created_from <= created_to (если оба заданы и корректны)
		cf, cfok := parseRFC3339(q.Get("created_from"))
		ct, ctok := parseRFC3339(q.Get("created_to"))
		if cfok && ctok && cf.After(ct) {
			v = append(v, render.Violation{
				Field: "created_from/created_to", Reason: "created_from must be <= created_to",
			})
		}
		// updated_from <= updated_to (если оба заданы и корректны)
		uf, ufok := parseRFC3339(q.Get("updated_from"))
		ut, utok := parseRFC3339(q.Get("updated_to"))
		if ufok && utok && uf.After(ut) {
			v = append(v, render.Violation{
				Field: "updated_from/updated_to", Reason: "updated_from must be <= updated_to",
			})
		}

		if len(v) > 0 {
			render.Validation(w, r, "INVALID", "invalid search parameters", v)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseInt(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func parseRFC3339(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	return t, err == nil
}
