package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/ivankudzin/tgapp/backend/internal/app/apiapp"
	"github.com/ivankudzin/tgapp/backend/internal/config"
)

func TestHealthz(t *testing.T) {
	cfg := config.Default()
	cfg.HTTP.Addr = ":0"

	app, err := apiapp.New(context.Background(), cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	ts := httptest.NewServer(app.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("get healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", resp.StatusCode, http.StatusOK)
	}

	var payload struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.OK {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
