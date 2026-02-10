package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	matchessvc "github.com/ivankudzin/tgapp/backend/internal/services/matches"
)

func TestReportReturnsTempUnavailableOnRedisError(t *testing.T) {
	svc := matchessvc.NewService(matchessvc.Dependencies{
		ReportRateStore:   reportRateStoreStub{err: errors.New("redis unavailable")},
		ReportMaxPer10Min: 3,
	})
	h := NewMatchesHandler(svc)

	body, err := json.Marshal(map[string]any{
		"target_id": 202,
		"reason":    "spam",
		"details":   "too many links",
	})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/report", bytes.NewReader(body))
	req = req.WithContext(authsvc.WithIdentity(context.Background(), authsvc.Identity{
		UserID: 101,
		SID:    "sid-101",
		Role:   "user",
	}))

	rr := httptest.NewRecorder()
	h.Report(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusServiceUnavailable)
	}

	var payload struct {
		Code          string `json:"code"`
		RetryAfterSec int64  `json:"retry_after_sec"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "TEMP_UNAVAILABLE" {
		t.Fatalf("unexpected error code: got %q want %q", payload.Code, "TEMP_UNAVAILABLE")
	}
	if payload.RetryAfterSec != 10 {
		t.Fatalf("unexpected retry_after_sec: got %d want %d", payload.RetryAfterSec, 10)
	}
}

type reportRateStoreStub struct {
	err error
}

func (s reportRateStoreStub) IncrementWindow(context.Context, string, time.Duration) (int64, time.Duration, error) {
	return 0, 0, s.err
}
