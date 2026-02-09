package errors

import (
	"encoding/json"
	"net/http"
	"time"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RateLimitError struct {
	Code          string     `json:"code"`
	Message       string     `json:"message"`
	RetryAfterSec int64      `json:"retry_after_sec"`
	CooldownUntil *time.Time `json:"cooldown_until"`
}

func Write(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
