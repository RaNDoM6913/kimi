package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/ivankudzin/tgapp/backend/internal/app/apiapp"
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

	app, err := apiapp.New(ctx, cfg, log)
	if err != nil {
		log.Fatal("create api app", zap.Error(err))
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			log.Error("shutdown api app", zap.Error(err))
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("api server failed", zap.Error(err))
		}
	}
}
