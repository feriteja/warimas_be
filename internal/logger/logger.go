package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Init initializes zap logger depending on the environment.
func Init(env string) {
	var cfg zap.Config

	if env == "production" {
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "json"
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.MessageKey = "message"
		cfg.EncoderConfig.LevelKey = "level"
		cfg.EncoderConfig.CallerKey = "caller"
		cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.OutputPaths = []string{"stdout"}
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Build logger
	var err error
	log, err = cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
}

// L returns the global logger.
func L() *zap.Logger {
	if log == nil {
		Init(os.Getenv("APP_ENV"))
	}
	return log
}

// Sync flushes logs.
func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}
