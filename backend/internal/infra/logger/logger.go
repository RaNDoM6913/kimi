package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"

	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)

	return cfg.Build()
}
