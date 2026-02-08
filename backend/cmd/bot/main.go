package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/ivankudzin/tgapp/backend/internal/app/botapp"
	"github.com/ivankudzin/tgapp/backend/internal/config"
	"github.com/ivankudzin/tgapp/backend/internal/infra/logger"
)

func main() {
	cfgPath := os.Getenv("APP_CONFIG")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		panic(err)
	}

	log, err := logger.New(cfg.Log.Level)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := botapp.New(ctx, cfg, log)
	if err != nil {
		log.Fatal("create bot app", zap.Error(err))
	}
	defer app.Close()

	if err := app.Run(ctx); err != nil {
		log.Fatal("bot app failed", zap.Error(err))
	}
}
