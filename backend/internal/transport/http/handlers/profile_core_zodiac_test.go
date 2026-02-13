package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	profilesvc "github.com/ivankudzin/tgapp/backend/internal/services/profiles"
)

type zodiacCaptureProfileStore struct {
	lastZodiac string
}

func (s *zodiacCaptureProfileStore) SaveCore(
	_ context.Context,
	_ int64,
	_ time.Time,
	_ string,
	_ string,
	_ string,
	_ string,
	_ int,
	_ string,
	zodiac string,
	_ []string,
	_ []string,
	_ bool,
) error {
	s.lastZodiac = zodiac
	return nil
}

func TestProfileCoreComputesAndSavesZodiac(t *testing.T) {
	store := &zodiacCaptureProfileStore{}
	service := profilesvc.NewService(store)
	handler := NewProfileHandler(service)

	body := `{
		"birthdate":"1995-03-12",
		"gender":"male",
		"looking_for":"female",
		"occupation":"engineer",
		"education":"higher",
		"height_cm":180,
		"eye_color":"brown",
		"languages":["ru","en"],
		"goals":["relationship"]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/profile/core", strings.NewReader(body))
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 42,
		SID:    "sid-42",
		Role:   "USER",
	}))
	rr := httptest.NewRecorder()

	handler.Core(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}
	if store.lastZodiac != "pisces" {
		t.Fatalf("unexpected saved zodiac: got %q want %q", store.lastZodiac, "pisces")
	}
}
