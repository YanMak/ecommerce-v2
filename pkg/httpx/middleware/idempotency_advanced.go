package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
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

// Конфиг идемпотентности.
type IdemConfig struct {
	TTL              time.Duration // TTL кэша ответа (например, 30m)
	LockTTL          time.Duration // TTL блокировки (должен покрывать worst-case времени операции, напр. 60s)
	MaxBodyBytes     int64         // лимит буферизации тела запроса (1MB по умолчанию)
	WaitForResult    time.Duration // ПРИ конфликте: сколько ждём появления кэша вместо 409 (напр. 500ms). 0 = сразу 409
	WaitPollInterval time.Duration // шаг опроса кэша при ожидании (напр. 100ms)
}

type cachedResp struct {
	Status      int               `json:"status"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        []byte            `json:"body"`
	Fingerprint string            `json:"fp"`
}

const luaUnlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
else
  return 0
end
`

// Idempotency — мидлварь идемпотентности для мутирующих методов (POST/PUT/PATCH/DELETE).
// Ключ берётся из заголовка "Idempotency-Key".
func Idempotency(rdb *redis.Client, cfg IdemConfig) func(http.Handler) http.Handler {
	if rdb == nil || cfg.TTL <= 0 || cfg.LockTTL <= 0 {
		// no-op
		return func(next http.Handler) http.Handler { return next }
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 1 << 20 // 1MB
	}
	if cfg.WaitPollInterval <= 0 {
		cfg.WaitPollInterval = 100 * time.Millisecond
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
			dataKey := "idem:data:" + route + ":" + r.Method + ":" + key
			lockKey := "idem:lock:" + route + ":" + r.Method + ":" + key

			// Буферизуем тело (для отпечатка) и восстанавливаем r.Body
			fp, body, okBody := fingerprintRequest(r, cfg.MaxBodyBytes)
			if okBody {
				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			// Короткий контекст для операций Redis
			rlim := 120 * time.Millisecond
			//rlim := 1200 * time.Second

			// 1) Попытка отдать из кэша
			{
				ctx1, cancel1 := context.WithTimeout(r.Context(), rlim)
				got := tryGetCached(ctx1, rdb, dataKey)
				cancel1()
				if got != nil {
					// Проверка reuse ключа с другим телом
					if got.Fingerprint == fp || fp == "" {
						writeCached(w, key, got)
						if span := trace.SpanFromContext(r.Context()); span != nil {
							span.SetAttributes(attribute.Bool("idem.cache_hit", true))
						}
						return
					}
					http.Error(w, "Idempotency-Key conflict", http.StatusConflict)
					return
				}
			}

			// 2) Пытаемся взять лок (с токеном)
			token := randToken()
			var gotLock bool
			{
				ctx2, cancel2 := context.WithTimeout(r.Context(), rlim)
				ok, _ := rdb.SetNX(ctx2, lockKey, token, cfg.LockTTL).Result()
				cancel2()
				gotLock = ok
			}

			if !gotLock {
				// Лок уже занят: попробуем немного подождать появления результата (если включено)
				if cfg.WaitForResult > 0 {
					if span := trace.SpanFromContext(r.Context()); span != nil {
						span.SetAttributes(attribute.Bool("idem.wait_on_conflict", true))
					}
					deadline := time.Now().Add(cfg.WaitForResult)
					for time.Now().Before(deadline) {
						// Маленькая задержка
						time.Sleep(cfg.WaitPollInterval)
						ctxPoll, cancelPoll := context.WithTimeout(r.Context(), rlim)
						got := tryGetCached(ctxPoll, rdb, dataKey)
						cancelPoll()
						if got != nil {
							// Нашли готовый результат — отдаём
							if got.Fingerprint == fp || fp == "" {
								writeCached(w, key, got)
								if span := trace.SpanFromContext(r.Context()); span != nil {
									span.SetAttributes(attribute.Bool("idem.cache_hit_after_wait", true))
								}
								return
							}
							http.Error(w, "Idempotency-Key conflict", http.StatusConflict)
							return
						}
					}
				}
				// Не дождались — конфликт (подсказываем повторить позже)
				w.Header().Set("Retry-After", "1")
				http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
				return
			}
			// Мы — владелец лока
			if span := trace.SpanFromContext(r.Context()); span != nil {
				span.SetAttributes(attribute.Bool("idem.lock_owner", true))
			}
			defer func() {
				// безопасное снятие лока по токену
				ctxU, cancelU := context.WithTimeout(context.Background(), rlim)
				defer cancelU()
				_, _ = rdb.Eval(ctxU, luaUnlockScript, []string{lockKey}, token).Result()
			}()

			// 3) Выполняем реальный хендлер, буферизуя ответ
			bw := newBufferingWriter(w, cfg.MaxBodyBytes)
			next.ServeHTTP(bw, r)

			// 4) Кэшируем ТОЛЬКО 2xx-ответы
			if bw.status >= 200 && bw.status < 300 && !bw.overflow {
				cr := &cachedResp{
					Status:      bw.status,
					ContentType: bw.ct,
					Headers:     pickHeaders(bw.Header()),
					Body:        bw.buf.Bytes(),
					Fingerprint: fp,
				}
				// Фоновая запись
				go func() {
					ctx3, cancel3 := context.WithTimeout(context.Background(), rlim)
					defer cancel3()
					saveCached(ctx3, rdb, dataKey, cr, cfg.TTL)
					// Подстрахуем: поставим TTL, если его вдруг нет
					_ = rdb.ExpireNX(ctx3, dataKey, cfg.TTL).Err()
				}()
				w.Header().Set("Idempotency-Key", key)
				w.Header().Set("Idempotency-Cache", "miss-store")
			} else if bw.overflow {
				// большой ответ — сознательно не кэшируем
				w.Header().Set("Idempotency-Cache", "skip-oversize")
			}
		})
	}
}

// --- вспомогательное ниже (без изменений по сути, но импортован strconv) ---

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
	lim := io.LimitedReader{R: r.Body, N: max + 1}
	b, _ := io.ReadAll(&lim)
	if int64(len(b)) > max {
		return "", b[:max], true
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), b, true
}

type bufferingWriter struct {
	http.ResponseWriter
	status   int
	buf      bytes.Buffer
	ct       string
	max      int64
	wrote    int64
	overflow bool
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
	// пробуем буферизовать до лимита; если вышли — отмечаем overflow
	if bw.wrote < bw.max {
		n := int64(len(p))
		toCopy := n
		if bw.wrote+n > bw.max {
			toCopy = bw.max - bw.wrote
			bw.overflow = true
		}
		if toCopy > 0 {
			bw.buf.Write(p[:toCopy])
			bw.wrote += toCopy
		} else {
			bw.overflow = true
		}
	} else {
		bw.overflow = true
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

func randToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func pickHeaders(h http.Header) map[string]string {
	out := make(map[string]string, 2)
	if v := h.Get("Content-Type"); v != "" {
		out["Content-Type"] = v
	}
	// при желании добавь ещё whitelisted заголовки ответа
	return out
}
