package query

import (
	"net/url"
	"strconv"
	"time"
)

// StringPtr: вернёт *s, если ключ есть и не пустой.
func StringPtr(q url.Values, key string) *string {
	if s := q.Get(key); s != "" {
		return &s
	}
	return nil
}

func Int64Ptr(q url.Values, key string) *int64 {
	if v := q.Get(key); v != "" {
		if x, err := strconv.ParseInt(v, 10, 64); err == nil {
			return &x
		}
	}
	return nil
}

func BoolPtr(q url.Values, key string) *bool {
	if v := q.Get(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return &b
		}
	}
	return nil
}

// TimePtrRFC3339: ожидает RFC3339 ("2025-09-09T12:34:56Z").
// При желании потом добавим разбор "YYYY-MM-DD".
func TimePtrRFC3339(q url.Values, key string) *time.Time {
	if v := q.Get(key); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return &t
		}
	}
	return nil
}

// IntDefault: читает int, иначе — def.
func IntDefault(q url.Values, key string, def int) int {
	if v := q.Get(key); v != "" {
		if x, err := strconv.Atoi(v); err == nil {
			return x
		}
	}
	return def
}
