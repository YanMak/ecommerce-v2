package log

import "go.uber.org/zap"

// NewProduction — готовый prod-конфиг (JSON, уровни, семплинг).
func NewProduction() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	// при желании можно подкрутить cfg.EncoderConfig / Sampling
	return cfg.Build()
}
