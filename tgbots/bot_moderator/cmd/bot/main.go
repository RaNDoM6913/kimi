package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"bot_moderator/internal/app"
	"bot_moderator/internal/config"
	loginfra "bot_moderator/internal/infra/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	logger := loginfra.New(cfg.LogLevel)

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("create app", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("bot starting")
	if err := application.Run(ctx); err != nil {
		logger.Error("bot stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("bot stopped")
}
