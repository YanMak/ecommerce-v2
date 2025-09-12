package config

import (
	"os"
	"strconv"
	"time"
)

func Str(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func Dur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
func Int(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
