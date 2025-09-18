package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewProduction — готовый prod-конфиг (JSON, уровни, семплинг).
func NewProduction() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	// при желании можно подкрутить cfg.EncoderConfig / Sampling
	return cfg.Build()
}

// initLogger.go (или где инициализируешь zap)
func NewLogger(devLogFile string) (*zap.Logger, error) {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	enc := zapcore.NewJSONEncoder(encCfg)

	lvl := zap.NewAtomicLevelAt(zap.InfoLevel)

	// всегда пишем в stdout
	syncers := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}

	// если задан файл — дублируем туда (только для DEV/Debug)
	if p := os.Getenv("DEV_LOG_FILE"); p != "" || devLogFile != "" {
		if p == "" && devLogFile != "" {
			p = devLogFile
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			syncers = append(syncers, zapcore.AddSync(f))
		}
	}

	core := zapcore.NewCore(enc, zapcore.NewMultiWriteSyncer(syncers...), lvl)
	lg := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	return lg, nil
}
