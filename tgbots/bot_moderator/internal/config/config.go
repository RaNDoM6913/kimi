package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken           string
	OwnerTGID          int64
	LogLevel           string
	PollTimeoutSeconds int
	DatabaseURL        string
	RedisAddr          string
	S3Endpoint         string
	S3AccessKey        string
	S3SecretKey        string
	S3UseSSL           bool
	S3Bucket           string
}

func Load() (Config, error) {
	ownerTGID, err := getInt64([]string{"OWNER_TG_ID", "owner_tg_id"}, 0)
	if err != nil {
		return Config{}, err
	}

	pollTimeout, err := getInt([]string{"POLL_TIMEOUT_SECONDS"}, 30)
	if err != nil {
		return Config{}, err
	}

	s3UseSSL, err := getBool([]string{"S3_USE_SSL"}, false)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		BotToken:           strings.TrimSpace(os.Getenv("BOT_TOKEN")),
		OwnerTGID:          ownerTGID,
		LogLevel:           getString("LOG_LEVEL", "info"),
		PollTimeoutSeconds: pollTimeout,
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		RedisAddr:          getString("REDIS_ADDR", "localhost:6379"),
		S3Endpoint:         getString("S3_ENDPOINT", ""),
		S3AccessKey:        getString("S3_ACCESS_KEY", ""),
		S3SecretKey:        getString("S3_SECRET_KEY", ""),
		S3UseSSL:           s3UseSSL,
		S3Bucket:           getString("S3_BUCKET", ""),
	}

	if cfg.PollTimeoutSeconds <= 0 {
		cfg.PollTimeoutSeconds = 30
	}

	return cfg, nil
}

func getString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getInt64(keys []string, fallback int64) (int64, error) {
	raw, key := getFirstDefined(keys)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}

func getInt(keys []string, fallback int) (int, error) {
	raw, key := getFirstDefined(keys)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}

func getBool(keys []string, fallback bool) (bool, error) {
	raw, key := getFirstDefined(keys)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}

func getFirstDefined(keys []string) (string, string) {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value, key
		}
	}
	if len(keys) == 0 {
		return "", ""
	}
	return "", keys[0]
}
