package middleware

import (
	"net/http"
	"time"

	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"
)

const HeaderRequestID = "X-Request-Id"

// WithRequestID: берёт X-Request-Id из запроса (или генерит),
// кладёт его в контекст и возвращает в ответ.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if id == "" {
			// можно заменить на uuid по желанию
			//id = r.Context().Value("req-start-ts").(string) // или просто time.Now().UTC().Format(...)
			// для краткости опустим генерацию — подставь свою
			id = time.Now().UTC().Format("20060102T150405.000000000Z07:00") // или uuid
		}
		ctx := tctx.WithRequestID(r.Context(), id)
		w.Header().Set(HeaderRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
