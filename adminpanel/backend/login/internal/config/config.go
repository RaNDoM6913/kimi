package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerAddr         string
	PostgresDSN        string
	JWTSecret          string
	JWTTTL             time.Duration
	SessionIdleTimeout time.Duration
	SessionMaxLifetime time.Duration
	TOTPSecretKey      string
	TelegramBotToken   string
	TelegramAuthMaxAge time.Duration
	ChallengeTTL       time.Duration
	TOTPSetupTTL       time.Duration
	MaxFailedAttempts  int
	LockDuration       time.Duration
	Issuer             string
	BootstrapKey       string
	DevMode            bool
}

func Load() (Config, error) {
	sessionMaxLifetime := getDuration("LOGIN_SESSION_MAX_LIFETIME", 12*time.Hour)
	cfg := Config{
		ServerAddr:         getString("LOGIN_HTTP_ADDR", ":8082"),
		PostgresDSN:        strings.TrimSpace(os.Getenv("LOGIN_POSTGRES_DSN")),
		JWTSecret:          strings.TrimSpace(os.Getenv("LOGIN_JWT_SECRET")),
		JWTTTL:             getDuration("LOGIN_JWT_TTL", sessionMaxLifetime),
		SessionIdleTimeout: getDuration("LOGIN_SESSION_IDLE_TIMEOUT", 30*time.Minute),
		SessionMaxLifetime: sessionMaxLifetime,
		TOTPSecretKey:      strings.TrimSpace(os.Getenv("LOGIN_TOTP_SECRET_KEY")),
		TelegramBotToken:   strings.TrimSpace(os.Getenv("LOGIN_TELEGRAM_BOT_TOKEN")),
		TelegramAuthMaxAge: getDuration("LOGIN_TELEGRAM_AUTH_MAX_AGE", 5*time.Minute),
		ChallengeTTL:       getDuration("LOGIN_CHALLENGE_TTL", 10*time.Minute),
		TOTPSetupTTL:       getDuration("LOGIN_TOTP_SETUP_TTL", 10*time.Minute),
		MaxFailedAttempts:  getInt("LOGIN_MAX_FAILED_ATTEMPTS", 5),
		LockDuration:       getDuration("LOGIN_LOCK_DURATION", 15*time.Minute),
		Issuer:             getString("LOGIN_2FA_ISSUER", "Heartbeat Admin"),
		BootstrapKey:       strings.TrimSpace(os.Getenv("LOGIN_BOOTSTRAP_KEY")),
		DevMode:            getBool("LOGIN_DEV_MODE", false),
	}

	var missing []string
	if cfg.PostgresDSN == "" {
		missing = append(missing, "LOGIN_POSTGRES_DSN")
	}
	if cfg.JWTSecret == "" {
		missing = append(missing, "LOGIN_JWT_SECRET")
	}
	if cfg.TOTPSecretKey == "" {
		missing = append(missing, "LOGIN_TOTP_SECRET_KEY")
	}
	if !cfg.DevMode && cfg.TelegramBotToken == "" {
		missing = append(missing, "LOGIN_TELEGRAM_BOT_TOKEN")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}

	if cfg.MaxFailedAttempts < 1 {
		cfg.MaxFailedAttempts = 5
	}
	if cfg.SessionIdleTimeout <= 0 {
		cfg.SessionIdleTimeout = 30 * time.Minute
	}
	if cfg.SessionMaxLifetime <= 0 {
		cfg.SessionMaxLifetime = 12 * time.Hour
	}
	if cfg.SessionIdleTimeout > cfg.SessionMaxLifetime {
		cfg.SessionIdleTimeout = cfg.SessionMaxLifetime
	}
	if cfg.JWTTTL <= 0 || cfg.JWTTTL > cfg.SessionMaxLifetime {
		cfg.JWTTTL = cfg.SessionMaxLifetime
	}

	return cfg, nil
}

func getString(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	if d <= 0 {
		return fallback
	}
	return d
}

func getInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}
