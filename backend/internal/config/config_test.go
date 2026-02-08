package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesDefaultsAndYAMLOverrides(t *testing.T) {
	clearConfigEnv(t)

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")
	yaml := `
remote:
  limits:
    free_likes_per_day: 99
  ads_inject:
    free: 11
  filters:
    radius_default_km: 8
  goals_mode: weighted_soft
  boost:
    duration: 45m
  me_defaults:
    timezone: UTC
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Remote.Limits.FreeLikesPerDay != 99 {
		t.Fatalf("unexpected free likes/day: %d", cfg.Remote.Limits.FreeLikesPerDay)
	}
	if cfg.Remote.AdsInject.FreeEvery != 11 {
		t.Fatalf("unexpected ads free inject: %d", cfg.Remote.AdsInject.FreeEvery)
	}
	if cfg.Remote.Filters.RadiusDefaultKM != 8 {
		t.Fatalf("unexpected default radius: %d", cfg.Remote.Filters.RadiusDefaultKM)
	}
	if cfg.Remote.GoalsMode != "weighted_soft" {
		t.Fatalf("unexpected goals mode: %s", cfg.Remote.GoalsMode)
	}
	if cfg.Remote.Boost.Duration.String() != "45m0s" {
		t.Fatalf("unexpected boost duration: %s", cfg.Remote.Boost.Duration.String())
	}
	if cfg.Remote.MeDefaults.Timezone != "UTC" {
		t.Fatalf("unexpected timezone: %s", cfg.Remote.MeDefaults.Timezone)
	}

	if !cfg.Remote.Limits.PlusUnlimitedUI {
		t.Fatalf("plus_unlimited_ui default should stay true")
	}
	if cfg.Remote.AdsInject.PlusEvery != 37 {
		t.Fatalf("ads plus default should stay 37")
	}
}

func TestLoadDefaultsWhenFileMissing(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("load config with missing file: %v", err)
	}

	if cfg.Remote.Limits.FreeLikesPerDay != 35 {
		t.Fatalf("unexpected default free likes/day: %d", cfg.Remote.Limits.FreeLikesPerDay)
	}
	if cfg.Remote.Filters.AgeMin != 18 || cfg.Remote.Filters.AgeMax != 30 {
		t.Fatalf("unexpected age defaults: %d-%d", cfg.Remote.Filters.AgeMin, cfg.Remote.Filters.AgeMax)
	}
	if cfg.Remote.Boost.Duration.String() != "30m0s" {
		t.Fatalf("unexpected default boost duration: %s", cfg.Remote.Boost.Duration.String())
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_ENV",
		"HTTP_ADDR",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"LOG_LEVEL",
		"POSTGRES_DSN",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"REDIS_DB",
		"S3_ENDPOINT",
		"S3_ACCESS_KEY",
		"S3_SECRET_KEY",
		"S3_BUCKET",
		"S3_USE_SSL",
		"JWT_SECRET",
		"JWT_ACCESS_TTL",
		"REFRESH_TTL",
		"BOT_TOKEN",
		"BOT_CLEANUP_INTERVAL",
		"BOT_CIRCLE_RETENTION",
	} {
		t.Setenv(key, "")
	}
}
