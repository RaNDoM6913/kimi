package config

import "testing"

func TestLoadAdminDefaults(t *testing.T) {
	t.Setenv("OWNER_TG_ID", "")
	t.Setenv("owner_tg_id", "")
	t.Setenv("POLL_TIMEOUT_SECONDS", "")
	t.Setenv("ADMIN_API_URL", "")
	t.Setenv("ADMIN_BOT_TOKEN", "")
	t.Setenv("ADMIN_MODE", "")
	t.Setenv("ADMIN_HTTP_TIMEOUT_SECONDS", "")
	t.Setenv("S3_USE_SSL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.AdminMode != "dual" {
		t.Fatalf("expected admin mode dual, got %q", cfg.AdminMode)
	}
	if cfg.AdminHTTPTimeout != 8 {
		t.Fatalf("expected default admin timeout 8, got %d", cfg.AdminHTTPTimeout)
	}
	if cfg.IsHTTPEnabled() {
		t.Fatal("expected http disabled by default")
	}
	if !cfg.IsDualMode() {
		t.Fatal("expected dual mode by default")
	}
}

func TestLoadAdminModeNormalization(t *testing.T) {
	t.Setenv("OWNER_TG_ID", "")
	t.Setenv("owner_tg_id", "")
	t.Setenv("POLL_TIMEOUT_SECONDS", "")
	t.Setenv("S3_USE_SSL", "")

	t.Setenv("ADMIN_MODE", "HTTP")
	t.Setenv("ADMIN_API_URL", "http://127.0.0.1:8080")
	t.Setenv("ADMIN_BOT_TOKEN", "token")
	t.Setenv("ADMIN_HTTP_TIMEOUT_SECONDS", "12")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.AdminMode != "http" {
		t.Fatalf("expected normalized mode http, got %q", cfg.AdminMode)
	}
	if cfg.AdminHTTPTimeout != 12 {
		t.Fatalf("expected timeout 12, got %d", cfg.AdminHTTPTimeout)
	}
	if !cfg.IsHTTPEnabled() {
		t.Fatal("expected http enabled")
	}
	if cfg.IsDualMode() {
		t.Fatal("expected non-dual mode for http")
	}
}

func TestIsDualModeForUnknownValue(t *testing.T) {
	cfg := Config{AdminMode: "invalid"}
	if !cfg.IsDualMode() {
		t.Fatal("expected unknown mode to normalize to dual")
	}
}
