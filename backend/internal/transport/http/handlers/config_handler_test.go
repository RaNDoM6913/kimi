package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ivankudzin/tgapp/backend/internal/config"
)

func TestConfigHandlerResponseShape(t *testing.T) {
	remote := config.Default().Remote
	h := NewConfigHandler(remote)

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rr := httptest.NewRecorder()
	h.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	requireObjectKey(t, raw, "limits")
	requireObjectKey(t, raw, "antiabuse")
	requireObjectKey(t, raw, "ads_inject")
	requireObjectKey(t, raw, "filters")
	requireObjectKey(t, raw, "goals_mode")
	requireObjectKey(t, raw, "boost")
	requireObjectKey(t, raw, "cities")

	limits := raw["limits"].(map[string]interface{})
	if int(limits["free_likes_per_day"].(float64)) != 35 {
		t.Fatalf("unexpected free_likes_per_day: %v", limits["free_likes_per_day"])
	}

	plus := limits["plus"].(map[string]interface{})
	if plus["unlimited_ui"].(bool) != true {
		t.Fatalf("unexpected plus.unlimited_ui: %v", plus["unlimited_ui"])
	}

	rate := plus["rate_limits"].(map[string]interface{})
	if int(rate["per_minute"].(float64)) <= 0 || int(rate["per_10sec"].(float64)) <= 0 {
		t.Fatalf("unexpected plus.rate_limits: %+v", rate)
	}

	antiabuse := raw["antiabuse"].(map[string]interface{})
	if int(antiabuse["like_max_per_sec"].(float64)) != 2 {
		t.Fatalf("unexpected antiabuse.like_max_per_sec: %v", antiabuse["like_max_per_sec"])
	}
	if int(antiabuse["report_max_10m"].(float64)) != 3 {
		t.Fatalf("unexpected antiabuse.report_max_10m: %v", antiabuse["report_max_10m"])
	}
	if int(antiabuse["new_device_risk_weight"].(float64)) != 3 {
		t.Fatalf("unexpected antiabuse.new_device_risk_weight: %v", antiabuse["new_device_risk_weight"])
	}
	steps := antiabuse["cooldown_steps_sec"].([]interface{})
	if len(steps) == 0 {
		t.Fatalf("unexpected antiabuse.cooldown_steps_sec: %v", antiabuse["cooldown_steps_sec"])
	}

	boost := raw["boost"].(map[string]interface{})
	if boost["duration"].(string) != "30m" {
		t.Fatalf("unexpected boost.duration: %v", boost["duration"])
	}

	cities := raw["cities"].([]interface{})
	if len(cities) != 6 {
		t.Fatalf("unexpected cities length: %d", len(cities))
	}
}

func requireObjectKey(t *testing.T, m map[string]interface{}, key string) {
	t.Helper()
	if _, ok := m[key]; !ok {
		t.Fatalf("missing key %q", key)
	}
}
