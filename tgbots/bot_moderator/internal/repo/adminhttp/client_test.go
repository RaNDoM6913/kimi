package adminhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientDoSetsRequiredHeaders(t *testing.T) {
	t.Parallel()

	const (
		token   = "bot-secret"
		actorID = int64(777001)
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Admin-Bot-Token"); got != token {
			t.Fatalf("unexpected X-Admin-Bot-Token: %q", got)
		}
		if got := r.Header.Get("X-Actor-Tg-Id"); got != "777001" {
			t.Fatalf("unexpected X-Actor-Tg-Id: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected Content-Type: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, token, time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status, response, err := client.do(context.Background(), http.MethodPost, "/admin/test", []byte(`{"x":1}`), actorID)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("unexpected status: %d", status)
	}
	if strings.TrimSpace(string(response)) != `{"ok":true}` {
		t.Fatalf("unexpected response body: %s", string(response))
	}
}

func TestClientDoClassifiesHTTPStatusFallback(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		status       int
		fallbackable bool
	}{
		{name: "server error", status: http.StatusInternalServerError, fallbackable: true},
		{name: "unauthorized", status: http.StatusUnauthorized, fallbackable: false},
		{name: "forbidden", status: http.StatusForbidden, fallbackable: false},
		{name: "validation", status: http.StatusBadRequest, fallbackable: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte("error"))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "token", time.Second)
			if err != nil {
				t.Fatalf("new client: %v", err)
			}

			_, _, err = client.do(context.Background(), http.MethodGet, "/admin/test", nil, 123)
			if err == nil {
				t.Fatalf("expected error for status %d", tc.status)
			}

			var reqErr *RequestError
			if !errors.As(err, &reqErr) {
				t.Fatalf("expected RequestError, got %T", err)
			}
			if reqErr.Fallbackable != tc.fallbackable {
				t.Fatalf("fallbackable mismatch: got=%v want=%v", reqErr.Fallbackable, tc.fallbackable)
			}
		})
	}
}

func TestClientDoClassifiesTimeoutAsFallbackable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", 30*time.Millisecond)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, _, err = client.do(context.Background(), http.MethodGet, "/admin/test", nil, 123)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !IsFallbackable(err) {
		t.Fatalf("expected timeout to be fallbackable, got err=%v", err)
	}
}
