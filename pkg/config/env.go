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
func Int64(key string, def int) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return int64(n)
		}
	}
	return int64(def)
}

func Float(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		fl, _ := strconv.ParseFloat(key, 64)
		return fl
	}
	return def
}
