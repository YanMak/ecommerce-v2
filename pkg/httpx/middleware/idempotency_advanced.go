package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Параметры: TTL ответа и TTL блокировки. maxBodyBytes — лимит читаемого тела (чтобы не ушатать память).
type IdemConfig struct {
	TTL          time.Duration
	LockTTL      time.Duration
	MaxBodyBytes int64 // напр. 1<<20 (1MB)
}

type cachedResp struct {
	Status      int               `json:"status"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers,omitempty"` // по минимуму
	Body        []byte            `json:"body"`
	Fingerprint string            `json:"fp"`
}

// Для POST/PUT/PATCH с заголовком Idempotency-Key:
// 1) пытаемся отдать из кеша
// 2) если нет — ставим lock; при гонке 409
// 3) выполняем хендлер, сохраняем (2xx) в кеш, снимаем lock
func Idempotency(rdb *redis.Client, cfg IdemConfig) func(http.Handler) http.Handler {
	if rdb == nil || cfg.TTL <= 0 || cfg.LockTTL <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 1 << 20 // 1MB
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutating(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			route := routePatternOrPath(r)
			fp, body, ok := fingerprintRequest(r, cfg.MaxBodyBytes) // body для восстановления r.Body
			if ok {
				r.Body = io.NopCloser(bytes.NewReader(body)) // восстановим, чтобы downstream прочитал
			}

			dataKey := "idem:data:" + route + ":" + r.Method + ":" + key
			lockKey := "idem:lock:" + route + ":" + r.Method + ":" + key

			// 0) Быстрый контекст для Redis
			rlim := 120 * time.Millisecond
			ctx, cancel := context.WithTimeout(r.Context(), rlim)
			defer cancel()

			// 1) Попытка отдать из кеша
			if got := tryGetCached(ctx, rdb, dataKey); got != nil {
				// Проверим, что отпечаток тот же (защита от повторного использования ключа с другим телом)
				if got.Fingerprint == fp || fp == "" {
					writeCached(w, key, got)
					if span := trace.SpanFromContext(r.Context()); span != nil {
						span.SetAttributes(attribute.Bool("idem.cache_hit", true))
					}
					return
				}
				// Иначе — конфликт: ключ повторно с другим содержимым
				http.Error(w, "Idempotency-Key conflict", http.StatusConflict)
				return
			}

			// 2) Пробуем взять lock
			okLock, _ := rdb.SetNX(ctx, lockKey, "1", cfg.LockTTL).Result()
			if !okLock {
				// Возможно, первая операция ещё в процессе. Можно вернуть 409 с Retry-After.
				w.Header().Set("Retry-After", "1")
				http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
				return
			}
			defer func() { _ = rdb.Del(context.Background(), lockKey).Err() }()

			// 3) Выполняем хендлер, буферизуя ответ (лимит — MaxBodyBytes)
			bw := newBufferingWriter(w, cfg.MaxBodyBytes)
			next.ServeHTTP(bw, r)

			// 4) Кешируем ТОЛЬКО успешные 2xx (можно расширить по желанию)
			if bw.status >= 200 && bw.status < 300 {
				cr := &cachedResp{
					Status:      bw.status,
					ContentType: bw.ct,
					Headers:     pickHeaders(bw.Header()),
					Body:        bw.buf.Bytes(),
					Fingerprint: fp,
				}
				// фоновая запись (не блокируем ответ клиенту)
				go func() {
					ctx2, cancel2 := context.WithTimeout(context.Background(), rlim)
					defer cancel2()
					saveCached(ctx2, rdb, dataKey, cr, cfg.TTL)
				}()
				w.Header().Set("Idempotency-Key", key)
				w.Header().Set("Idempotency-Cache", "miss-store")
			}
		})
	}
}

func isMutating(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func fingerprintRequest(r *http.Request, max int64) (string, []byte, bool) {
	if r.Body == nil {
		return "", nil, false
	}
	// читаем с ограничением
	lim := io.LimitedReader{R: r.Body, N: max + 1}
	b, _ := io.ReadAll(&lim)
	// если > max — не считаем fp, чтобы не хранить огромные тела
	if int64(len(b)) > max {
		return "", b[:max], true
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), b, true
}

type bufferingWriter struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
	ct     string
	max    int64
	wrote  int64
}

func newBufferingWriter(w http.ResponseWriter, max int64) *bufferingWriter {
	return &bufferingWriter{ResponseWriter: w, status: 200, max: max}
}

func (bw *bufferingWriter) WriteHeader(code int) {
	bw.status = code
	bw.ct = bw.Header().Get("Content-Type")
	bw.ResponseWriter.WriteHeader(code)
}

func (bw *bufferingWriter) Write(p []byte) (int, error) {
	// ограничим буфер
	if bw.wrote < bw.max {
		n := int64(len(p))
		toCopy := n
		if bw.wrote+n > bw.max {
			toCopy = bw.max - bw.wrote
		}
		bw.buf.Write(p[:toCopy])
		bw.wrote += toCopy
	}
	return bw.ResponseWriter.Write(p)
}

func tryGetCached(ctx context.Context, rdb *redis.Client, key string) *cachedResp {
	raw, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil
	}
	var cr cachedResp
	if json.Unmarshal(raw, &cr) == nil {
		return &cr
	}
	return nil
}

func saveCached(ctx context.Context, rdb *redis.Client, key string, cr *cachedResp, ttl time.Duration) {
	b, err := json.Marshal(cr)
	if err != nil {
		return
	}
	_ = rdb.Set(ctx, key, b, ttl).Err()
}

func writeCached(w http.ResponseWriter, idemKey string, cr *cachedResp) {
	for k, v := range cr.Headers {
		w.Header().Set(k, v)
	}
	if cr.ContentType != "" {
		w.Header().Set("Content-Type", cr.ContentType)
	}
	w.Header().Set("Idempotency-Key", idemKey)
	w.Header().Set("Idempotency-Cache", "hit")
	w.WriteHeader(cr.Status)
	_, _ = w.Write(cr.Body)
}

func pickHeaders(h http.Header) map[string]string {
	out := make(map[string]string, 2)
	if v := h.Get("Content-Type"); v != "" {
		out["Content-Type"] = v
	}
	// при желании добавь ещё whitelisted заголовки ответа
	return out
}
