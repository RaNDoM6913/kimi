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
geo:
  exact_retention_hours: 72
remote:
  limits:
    free_likes_per_day: 99
  antiabuse:
    like_max_min: 66
    report_max_10m: 5
    new_device_risk_weight: 9
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
	if cfg.Remote.AntiAbuse.LikeMaxPerMin != 66 {
		t.Fatalf("unexpected antiabuse like_max_min: %d", cfg.Remote.AntiAbuse.LikeMaxPerMin)
	}
	if cfg.Remote.AntiAbuse.ReportMaxPer10Min != 5 {
		t.Fatalf("unexpected antiabuse report_max_10m: %d", cfg.Remote.AntiAbuse.ReportMaxPer10Min)
	}
	if cfg.Remote.AntiAbuse.NewDeviceRiskWeight != 9 {
		t.Fatalf("unexpected antiabuse new_device_risk_weight: %d", cfg.Remote.AntiAbuse.NewDeviceRiskWeight)
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
	if cfg.Geo.ExactRetentionHours != 72 {
		t.Fatalf("unexpected geo.exact_retention_hours override: %d", cfg.Geo.ExactRetentionHours)
	}

	if !cfg.Remote.Limits.PlusUnlimitedUI {
		t.Fatalf("plus_unlimited_ui default should stay true")
	}
	if cfg.Remote.AdsInject.PlusEvery != 37 {
		t.Fatalf("ads plus default should stay 37")
	}
	if cfg.Remote.AntiAbuse.LikeMaxPerSec != 2 {
		t.Fatalf("antiabuse like_max_per_sec default should stay 2")
	}
	if cfg.Remote.AntiAbuse.ShadowRankMultiplier != 0.4 {
		t.Fatalf("antiabuse shadow_rank_multiplier default should stay 0.4")
	}
	if cfg.Admin.BotRole != "MODERATOR" {
		t.Fatalf("unexpected admin.bot_role default: %s", cfg.Admin.BotRole)
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
	if cfg.Remote.AntiAbuse.SuspectLikeThreshold != 8 {
		t.Fatalf("unexpected antiabuse suspect_like_threshold: %d", cfg.Remote.AntiAbuse.SuspectLikeThreshold)
	}
	if cfg.Remote.AntiAbuse.ReportMaxPer10Min != 3 {
		t.Fatalf("unexpected antiabuse report_max_10m default: %d", cfg.Remote.AntiAbuse.ReportMaxPer10Min)
	}
	if cfg.Remote.AntiAbuse.NewDeviceRiskWeight != 3 {
		t.Fatalf("unexpected antiabuse new_device_risk_weight default: %d", cfg.Remote.AntiAbuse.NewDeviceRiskWeight)
	}
	if len(cfg.Remote.AntiAbuse.CooldownStepsSec) != 5 {
		t.Fatalf("unexpected antiabuse cooldown steps length: %d", len(cfg.Remote.AntiAbuse.CooldownStepsSec))
	}
	if cfg.Admin.BotRole != "MODERATOR" {
		t.Fatalf("unexpected admin.bot_role default: %s", cfg.Admin.BotRole)
	}
	if cfg.Geo.ExactRetentionHours != 48 {
		t.Fatalf("unexpected geo.exact_retention_hours default: %d", cfg.Geo.ExactRetentionHours)
	}
}

func TestLoadRejectsMissingAdminBotTokenInProduction(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_ENV", "prod")

	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatalf("expected error when admin.bot_token is empty in production")
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
		"ADMIN_BOT_TOKEN",
		"ADMIN_BOT_ROLE",
		"GEO_EXACT_RETENTION_HOURS",
	} {
		t.Setenv(key, "")
	}
}
