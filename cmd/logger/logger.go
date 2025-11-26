package logger

import (
	"log"
	"project/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(config *config.Config) *zap.Logger {
	isDebug := config.Debug
	atom := zap.NewAtomicLevel()
	incodeCfg := zap.NewProductionEncoderConfig()
	var cfg zap.Config
	if isDebug {
		atom.SetLevel(zap.DebugLevel)
		incodeCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg = zap.Config{
			Encoding:      "console",
			Level:         atom,
			OutputPaths:   []string{"stdout", "logs/main.log"},
			EncoderConfig: incodeCfg,
		}
	} else {
		atom.SetLevel(zap.InfoLevel)
		cfg = zap.Config{
			Encoding:      "json",
			Level:         atom,
			OutputPaths:   []string{"stdout", "logs/main.log"},
			EncoderConfig: incodeCfg,
		}
	}

	logger, err := cfg.Build()
	if err != nil {
		log.Println(err)
	}
	return logger
}
