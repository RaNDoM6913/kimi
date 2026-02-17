package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ivankudzin/tgapp/adminpanel/backend/login/internal/config"
	"github.com/ivankudzin/tgapp/adminpanel/backend/login/internal/httpapi"
	"github.com/ivankudzin/tgapp/adminpanel/backend/login/internal/repo"
	"github.com/ivankudzin/tgapp/adminpanel/backend/login/internal/security"
	"github.com/ivankudzin/tgapp/adminpanel/backend/login/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	pool, err := repo.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("init postgres: %v", err)
	}
	defer pool.Close()

	users := repo.NewAdminUserRepo(pool)
	challenges := repo.NewChallengeRepo(pool)
	setupTokens := repo.NewTOTPSetupTokenRepo(pool)
	sessions := repo.NewSessionRepo(pool)
	tokenManager := security.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	secretCipher, err := security.NewSecretCipher(cfg.TOTPSecretKey)
	if err != nil {
		log.Fatalf("init totp secret cipher: %v", err)
	}
	authService := service.NewService(
		users,
		challenges,
		setupTokens,
		sessions,
		tokenManager,
		secretCipher,
		cfg.TelegramBotToken,
		cfg.TelegramAuthMaxAge,
		cfg.ChallengeTTL,
		cfg.TOTPSetupTTL,
		cfg.SessionIdleTimeout,
		cfg.SessionMaxLifetime,
		cfg.MaxFailedAttempts,
		cfg.LockDuration,
		cfg.Issuer,
		cfg.DevMode,
	)

	server := httpapi.NewServer(cfg.ServerAddr, authService, cfg.BootstrapKey)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("login api is listening on %s", cfg.ServerAddr)
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}

	log.Println("server stopped")
}
