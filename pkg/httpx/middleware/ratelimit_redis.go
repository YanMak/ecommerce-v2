package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RateLimitFixedWindow ограничивает N запросов за окно window.
// subjectFn выбирает «субъект» (пользователь/IP/тенант).
func RateLimitFixedWindow(rdb *redis.Client, limit int, window time.Duration, subjectFn func(*http.Request) string) func(http.Handler) http.Handler {
	if rdb == nil || limit <= 0 || window <= 0 {
		// no-op
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := routePatternOrPath(r)
			subj := "anon"
			if subjectFn != nil {
				if s := subjectFn(r); s != "" {
					subj = s
				}
			} else {
				subj = clientIP(r)
			}
			key := "rl:" + route + ":" + subj

			// Быстрый и короткий контекст для Redis
			ctx, cancel := context.WithTimeout(r.Context(), 80*time.Millisecond)
			defer cancel()

			// INCR; если первый раз — поставить EXPIRE=window
			n, err := rdb.Incr(ctx, key).Result()
			if err == nil && n == 1 {
				_ = rdb.Expire(ctx, key, window).Err()
			}
			// else {
			// 	res := rdb.Expire(ctx, key, 10*time.Second).Err()
			// 	fmt.Println("erro while rdb.Expire(ctx, key, 10*time.Nanosecond).Err()", res)
			// }

			span := trace.SpanFromContext(r.Context())

			// Ошибка Redis — fail-open (пропускаем), но отметим в трейс.
			if err != nil {
				if span != nil {
					span.SetAttributes(attribute.String("rl.error", err.Error()))
				}
				next.ServeHTTP(w, r)
				return
			}

			// Превышение лимита → 429 + Retry-After = TTL оставшегося окна (округляем вверх)
			if int(n) > limit {
				//ttl, _ := rdb.TTL(ctx, key).Result()
				ttl, err := rdb.TTL(ctx, key).Result()
				fmt.Println("error while ttl, err := rdb.TTL(ctx, key).Result()", err)
				after := int(ttl.Seconds())
				if after < 1 {
					after = int(window.Seconds())
				}
				if span != nil {
					span.SetAttributes(
						attribute.Bool("rate_limited", true),
						attribute.String("rl.route", route),
						attribute.String("rl.subject", subj),
					)
				}
				w.Header().Set("Retry-After", itoa(after))
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func itoa(x int) string {
	// без strconv импортов, чтобы middleware был лёгким
	return fmtInt(x)
}

// простая целочисленная конвертация (чтобы не тянуть strconv).
func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if sign != "" {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
