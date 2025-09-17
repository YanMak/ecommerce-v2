package middleware

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type RLScope string

const (
	RLScopeGlobal      RLScope = "global"
	RLScopeIP          RLScope = "ip"
	RLScopeRoute       RLScope = "route"
	RLScopeIPRoute     RLScope = "ip+route"
	RLScopeUser        RLScope = "user"
	RLScopeUserRoute   RLScope = "user+route"
	RLScopeAPIKey      RLScope = "apikey"
	RLScopeAPIKeyRoute RLScope = "apikey+route"
)

type RateLimitConfig struct {
	Rate      float64       // токенов в секунду, напр. 20
	Burst     int           // ёмкость бакета (сколько можно залить мгновенно), напр. 40
	Scope     RLScope       // "ip+route" по умолчанию
	KeyPrefix string        // префикс ключей в Redis, напр. "rl:tb:"
	TTL       time.Duration // ttl ключа (обычно >= Burst/Rate * 2)
	Wait      time.Duration // (опц.) подождать N перед отказом (не блокируем, просто одна попытка тикнуть бакет чуть позже)

	// НОВОЕ:
	UserIDHeader string // дефолт: "X-User-ID"
	APIKeyHeader string // дефолт: "X-API-Key"
}

type rlMetrics struct {
	allowed  *prometheus.CounterVec
	limited  *prometheus.CounterVec
	retrySec *prometheus.HistogramVec // НОВОЕ: сколько секунд ждать до следующего токена
}

func newRLMetrics(reg prometheus.Registerer) *rlMetrics {
	return &rlMetrics{
		allowed: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_rl_allowed_total",
				Help: "Allowed requests by token-bucket rate limiter.",
			},
			[]string{"scope", "route"},
		),
		limited: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_rl_limited_total",
				Help: "Limited (429) requests by token-bucket rate limiter.",
			},
			[]string{"scope", "route"},
		),
		retrySec: promauto.With(reg).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gateway_rl_retry_after_seconds",
				Help:    "Retry-After seconds suggested by rate limiter.",
				Buckets: prometheus.DefBuckets, // ок для старта
			},
			[]string{"scope", "route"},
		),
	}
}

// Lua — атомарный token bucket (float-токены, пополнение по времени)
// KEYS[1] = key
// ARGV[1]=now_ms, [2]=rate_per_sec, [3]=capacity, [4]=cost(обычно 1), [5]=ttl_sec
// return: {allowed(1/0), remaining_int, retry_after_sec}
const luaTokenBucket = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local cap  = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local ttl  = tonumber(ARGV[5])

local tokens = tonumber(redis.call('HGET', key, 'tokens'))
local ts     = tonumber(redis.call('HGET', key, 'ts'))

if tokens == nil or ts == nil then
  tokens = cap
  ts = now
else
  local delta = now - ts
  if delta > 0 then
    local add = delta * rate / 1000.0
    tokens = math.min(cap, tokens + add)
    ts = now
  end
end

local allowed = 0
local retry_sec = 0
if tokens >= cost then
  tokens = tokens - cost
  allowed = 1
else
  local need = cost - tokens
  retry_sec = math.ceil(need * 1.0 / rate)
end

redis.call('HSET', key, 'tokens', tokens, 'ts', ts)
redis.call('EXPIRE', key, ttl)

return {allowed, math.floor(tokens), retry_sec}
`

// RateLimitTokenBucket — глобальная мидлварь для chi.
func RateLimitTokenBucket(rdb *redis.Client, reg prometheus.Registerer, cfg RateLimitConfig) func(http.Handler) http.Handler {
	if rdb == nil || cfg.Rate <= 0 || cfg.Burst <= 0 {
		return func(next http.Handler) http.Handler { return next } // no-op
	}
	if cfg.Scope == "" {
		cfg.Scope = RLScopeIPRoute
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "rl:tb:"
	}
	// разумный TTL: минимум 2×до полного восстановления бакета, но не меньше 60s
	minTTL := time.Duration(float64(cfg.Burst) / cfg.Rate * 2.0 * float64(time.Second))
	if cfg.TTL <= 0 || cfg.TTL < minTTL {
		cfg.TTL = minTTL
		if cfg.TTL < 60*time.Second {
			cfg.TTL = 60 * time.Second
		}
	}

	metrics := newRLMetrics(reg)
	script := redis.NewScript(luaTokenBucket)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, route := rlKey(cfg, r)
			now := time.Now()
			rlim := 120 * time.Millisecond

			allowed, remain, retry := evalBucket(r.Context(), rdb, script, key, now, cfg, rlim)
			if !allowed && cfg.Wait > 0 {
				// мягкая попытка ещё раз через Wait (иногда спасает от фронтовых "шипов")
				time.Sleep(cfg.Wait)
				allowed, remain, retry = evalBucket(r.Context(), rdb, script, key, time.Now(), cfg, rlim)
			}
			// Заголовки по RateLimit (IETF): Limit, Remaining, Reset (сек до следующего токена)
			w.Header().Set("RateLimit-Limit", strconv.Itoa(cfg.Burst))
			w.Header().Set("RateLimit-Remaining", strconv.Itoa(remain))
			w.Header().Set("RateLimit-Reset", strconv.Itoa(retry))

			lblRoute := route
			if len(lblRoute) > 120 {
				lblRoute = hash(lblRoute) // чтобы не плодить длинные лейблы
			}
			shard := hash(key) // короткий хэш ключа (без утечки исходных значений)

			// OTel: помечаем текущий спан
			span := trace.SpanFromContext(r.Context())
			annotateRL(span, string(cfg.Scope), lblRoute, shard, allowed, remain, retry, cfg.Rate, cfg.Burst)

			if !allowed {
				metrics.limited.WithLabelValues(string(cfg.Scope), lblRoute)
				// было: .Inc()
				addCounterWithExemplar(metrics.limited.WithLabelValues(string(cfg.Scope), lblRoute), span)
				if retry <= 0 {
					retry = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retry))
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			//metrics.allowed.WithLabelValues(string(cfg.Scope), lblRoute).Inc()
			// allowed-путь
			addCounterWithExemplar(metrics.allowed.WithLabelValues(string(cfg.Scope), lblRoute), span)
			next.ServeHTTP(w, r)

		})
	}
}

func evalBucket(ctx context.Context, rdb *redis.Client, script *redis.Script, key string, now time.Time, cfg RateLimitConfig, rlim time.Duration) (allowed bool, remaining int, retrySec int) {
	ct, cancel := context.WithTimeout(ctx, rlim)
	defer cancel()

	res, err := script.Run(ct, rdb, []string{key},
		now.UnixMilli(),
		cfg.Rate,
		cfg.Burst,
		1, // cost per request
		int(cfg.TTL.Seconds()),
	).Result()
	if err != nil {
		// fail-open: лучше пустить трафик, чем сломать сервис из-за Redis
		return true, cfg.Burst, 0
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) != 3 {
		return true, cfg.Burst, 0
	}

	al, _ := asInt(arr[0])
	rm, _ := asInt(arr[1])
	rt, _ := asInt(arr[2])

	return al == 1, rm, rt
}

func rlKey(cfg RateLimitConfig, r *http.Request) (key string, route string) {
	route = routePatternOrPath(r)
	ip := clientIP(r)
	user := strings.TrimSpace(r.Header.Get(firstNonEmpty(cfg.UserIDHeader, "X-User-ID")))
	apikey := strings.TrimSpace(r.Header.Get(firstNonEmpty(cfg.APIKeyHeader, "X-API-Key")))

	switch cfg.Scope {
	case RLScopeGlobal:
		key = cfg.KeyPrefix + "global"
	case RLScopeIP:
		key = cfg.KeyPrefix + "ip:" + ip
	case RLScopeRoute:
		key = cfg.KeyPrefix + "route:" + route
	case RLScopeIPRoute:
		key = cfg.KeyPrefix + "ip:" + ip + "|route:" + route
	case RLScopeUser:
		if user == "" {
			user = "anon"
		}
		key = cfg.KeyPrefix + "user:" + user
	case RLScopeUserRoute:
		if user == "" {
			user = "anon"
		}
		key = cfg.KeyPrefix + "user:" + user + "|route:" + route
	case RLScopeAPIKey:
		if apikey == "" {
			apikey = "anon"
		}
		key = cfg.KeyPrefix + "key:" + apikey
	case RLScopeAPIKeyRoute:
		if apikey == "" {
			apikey = "anon"
		}
		key = cfg.KeyPrefix + "key:" + apikey + "|route:" + route
	default:
		key = cfg.KeyPrefix + "ip:" + ip + "|route:" + route
	}
	return key, route
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// func routePatternOrPath(r *http.Request) string {
// 	if rc := chi.RouteContext(r.Context()); rc != nil {
// 		if p := rc.RoutePattern(); p != "" {
// 			return p
// 		}
// 	}
// 	return r.URL.Path
// }

// func clientIP(r *http.Request) string {
// 	h := r.Header.Get("X-Forwarded-For")
// 	if h != "" {
// 		// берём первый
// 		p := strings.Split(h, ",")
// 		return strings.TrimSpace(p[0])
// 	}
// 	host, _, err := net.SplitHostPort(r.RemoteAddr)
// 	if err == nil && host != "" {
// 		return host
// 	}
// 	return r.RemoteAddr
// }

func asInt(v interface{}) (int, bool) {
	switch x := v.(type) {
	case int64:
		return int(x), true
	case int:
		return x, true
	case string:
		// не парсим — не нужно для Redis ответа
	}
	return 0, false
}

func hash(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:8])
}

func annotateRL(span trace.Span, scope, route, shard string, allowed bool, remaining, retry int, rate float64, burst int) {
	if span == nil {
		return
	}
	span.SetAttributes(
		attribute.String("rl.scope", scope),
		attribute.String("rl.route", route),     // это паттерн chi, стабильная кардинальность
		attribute.String("rl.key_shard", shard), // хэш ключа, без PII
		attribute.Bool("rl.allowed", allowed),
		attribute.Int("rl.remaining", remaining),
		attribute.Int("rl.retry_after", retry),
		attribute.Float64("rl.rate", rate),
		attribute.Int("rl.burst", burst),
	)
	if !allowed {
		span.AddEvent("rate_limited")
	}
}

// addCounterWithExemplar пытается добавить exemplar (trace_id) к counter; фолбэк — обычный Inc.
func addCounterWithExemplar(c prometheus.Counter, span trace.Span) {
	if span != nil {
		sc := span.SpanContext()
		if sc.HasTraceID() {
			if ea, ok := c.(prometheus.ExemplarAdder); ok {
				ea.AddWithExemplar(1, prometheus.Labels{"trace_id": sc.TraceID().String()})
				return
			}
		}
	}
	c.Inc()
}

// observeHistWithExemplar — то же для гистограммы.
func observeHistWithExemplar(h prometheus.Observer, span trace.Span, v float64) {
	if span != nil {
		sc := span.SpanContext()
		if sc.HasTraceID() {
			if eo, ok := h.(prometheus.ExemplarObserver); ok {
				eo.ObserveWithExemplar(v, prometheus.Labels{"trace_id": sc.TraceID().String()})
				return
			}
		}
	}
	h.Observe(v)
}
