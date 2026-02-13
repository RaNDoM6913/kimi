package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	feedsvc "github.com/ivankudzin/tgapp/backend/internal/services/feed"
	mediasvc "github.com/ivankudzin/tgapp/backend/internal/services/media"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
)

type candidateRepoStub struct {
	candidates map[int64]pgrepo.CandidateProfileRecord
}

func (s candidateRepoStub) GetViewerContext(context.Context, int64) (pgrepo.FeedViewerContext, error) {
	return pgrepo.FeedViewerContext{}, nil
}

func (s candidateRepoStub) ListCandidates(context.Context, pgrepo.FeedQuery) ([]pgrepo.FeedCandidate, error) {
	return nil, nil
}

func (s candidateRepoStub) GetCandidateProfile(_ context.Context, q pgrepo.CandidateProfileQuery) (pgrepo.CandidateProfileRecord, error) {
	if candidate, ok := s.candidates[q.CandidateUserID]; ok {
		return candidate, nil
	}
	return pgrepo.CandidateProfileRecord{}, pgrepo.ErrFeedCandidateNotFound
}

type candidateTelemetryStoreStub struct {
	events []pgrepo.EventWriteRecord
}

func (s *candidateTelemetryStoreStub) InsertBatch(_ context.Context, _ *int64, events []pgrepo.EventWriteRecord) error {
	s.events = append(s.events, events...)
	return nil
}

type candidateMediaStoreStub struct {
	photos map[int64][]mediasvc.PhotoRecord
}

func (s candidateMediaStoreStub) CreatePhoto(context.Context, int64, string) (mediasvc.PhotoRecord, error) {
	return mediasvc.PhotoRecord{}, fmt.Errorf("not implemented")
}

func (s candidateMediaStoreStub) ListActivePhotos(_ context.Context, userID int64) ([]mediasvc.PhotoRecord, error) {
	rows := s.photos[userID]
	out := make([]mediasvc.PhotoRecord, 0, len(rows))
	out = append(out, rows...)
	return out, nil
}

type candidateMediaStorageStub struct{}

func (candidateMediaStorageStub) EnsureBucket(context.Context) error {
	return nil
}

func (candidateMediaStorageStub) PutPhoto(context.Context, string, io.Reader, int64, string) error {
	return nil
}

func (candidateMediaStorageStub) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://signed.local/" + key, nil
}

func (candidateMediaStorageStub) Delete(context.Context, string) error {
	return nil
}

func TestCandidateProfileApprovedReturns200(t *testing.T) {
	service := feedsvc.NewService(candidateRepoStub{
		candidates: map[int64]pgrepo.CandidateProfileRecord{
			200: {
				UserID:      200,
				DisplayName: "Alice",
				Age:         25,
				Zodiac:      "aries",
				CityID:      "minsk",
				City:        "Minsk",
				Occupation:  "Designer",
				Education:   "BSU",
				HeightCM:    170,
				EyeColor:    "green",
				Languages:   []string{"ru", "en"},
				Goals:       []string{"relationship"},
				IsPlus:      false,
			},
		},
	}, feedsvc.Config{})
	store := &candidateTelemetryStoreStub{}
	telemetry := analyticsvc.NewService(store, analyticsvc.Config{MaxBatchSize: 100})
	handler := NewCandidateHandler(service, nil, telemetry)
	handler.now = func() time.Time {
		return time.Date(2026, 2, 10, 20, 0, 0, 0, time.UTC)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/candidates/200/profile", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 10,
		SID:    "sid-10",
		Role:   "USER",
	}))
	req = req.WithContext(withURLParam(req.Context(), "user_id", "200"))

	rr := httptest.NewRecorder()
	handler.Profile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	var payload dto.CandidateProfileResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.UserID != 200 || payload.Zodiac != "aries" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.Languages) != 2 || len(payload.Goals) != 1 {
		t.Fatalf("unexpected arrays in payload: %+v", payload)
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one candidate open event, got %d", len(store.events))
	}
	if store.events[0].Name != "candidate_profile_open" {
		t.Fatalf("unexpected event name: %q", store.events[0].Name)
	}
}

func TestCandidateProfileUnavailableReturns404(t *testing.T) {
	service := feedsvc.NewService(candidateRepoStub{
		candidates: map[int64]pgrepo.CandidateProfileRecord{
			100: {
				UserID:      100,
				DisplayName: "Approved",
				Age:         23,
				Zodiac:      "taurus",
				CityID:      "minsk",
				City:        "Minsk",
				Occupation:  "Dev",
				Education:   "BSU",
				HeightCM:    180,
				EyeColor:    "brown",
				Languages:   []string{"ru"},
				Goals:       []string{"relationship"},
			},
		},
	}, feedsvc.Config{})
	handler := NewCandidateHandler(service, nil, nil)

	cases := []struct {
		name string
		id   string
	}{
		{name: "pending", id: "101"},
		{name: "rejected", id: "102"},
		{name: "blocked", id: "103"},
		{name: "banned", id: "104"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/candidates/"+tc.id+"/profile", nil)
			req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
				UserID: 10,
				SID:    "sid-10",
				Role:   "USER",
			}))
			req = req.WithContext(withURLParam(req.Context(), "user_id", tc.id))

			rr := httptest.NewRecorder()
			handler.Profile(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestCandidatePhotosApprovedReturns200(t *testing.T) {
	service := feedsvc.NewService(candidateRepoStub{
		candidates: map[int64]pgrepo.CandidateProfileRecord{
			200: {
				UserID:      200,
				DisplayName: "Alice",
				Age:         25,
				Zodiac:      "aries",
				CityID:      "minsk",
				City:        "Minsk",
				Occupation:  "Designer",
				Education:   "BSU",
				HeightCM:    170,
				EyeColor:    "green",
				Languages:   []string{"ru", "en"},
				Goals:       []string{"relationship"},
			},
		},
	}, feedsvc.Config{})
	mediaService := mediasvc.NewService(candidateMediaStoreStub{
		photos: map[int64][]mediasvc.PhotoRecord{
			200: {
				{ID: 1, Position: 1, ObjectKey: "users/200/photos/1.jpg", CreatedAt: time.Date(2026, 2, 10, 9, 0, 0, 0, time.UTC)},
				{ID: 2, Position: 2, ObjectKey: "users/200/photos/2.jpg", CreatedAt: time.Date(2026, 2, 10, 9, 1, 0, 0, time.UTC)},
				{ID: 3, Position: 3, ObjectKey: "users/200/photos/3.jpg", CreatedAt: time.Date(2026, 2, 10, 9, 2, 0, 0, time.UTC)},
			},
		},
	}, candidateMediaStorageStub{})
	store := &candidateTelemetryStoreStub{}
	telemetry := analyticsvc.NewService(store, analyticsvc.Config{MaxBatchSize: 100})
	handler := NewCandidateHandler(service, mediaService, telemetry)

	req := httptest.NewRequest(http.MethodGet, "/v1/candidates/200/media/photos", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 10,
		SID:    "sid-10",
		Role:   "USER",
	}))
	req = req.WithContext(withURLParam(req.Context(), "user_id", "200"))

	rr := httptest.NewRecorder()
	handler.Photos(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	var payload dto.CandidatePhotosResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.UserID != 200 {
		t.Fatalf("unexpected user id: got %d want %d", payload.UserID, 200)
	}
	if len(payload.Photos) != 3 {
		t.Fatalf("unexpected photo count: got %d want %d", len(payload.Photos), 3)
	}
	for idx, photo := range payload.Photos {
		if photo.URL == "" {
			t.Fatalf("expected non-empty url for photo index %d", idx)
		}
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one candidate photos event, got %d", len(store.events))
	}
	if store.events[0].Name != "candidate_photos_fetch" {
		t.Fatalf("unexpected event name: %q", store.events[0].Name)
	}
}

func TestCandidatePhotosUnavailableReturns404(t *testing.T) {
	service := feedsvc.NewService(candidateRepoStub{
		candidates: map[int64]pgrepo.CandidateProfileRecord{
			100: {
				UserID:      100,
				DisplayName: "Approved",
				Age:         23,
				Zodiac:      "taurus",
				CityID:      "minsk",
				City:        "Minsk",
				Occupation:  "Dev",
				Education:   "BSU",
				HeightCM:    180,
				EyeColor:    "brown",
				Languages:   []string{"ru"},
				Goals:       []string{"relationship"},
			},
		},
	}, feedsvc.Config{})
	mediaService := mediasvc.NewService(candidateMediaStoreStub{
		photos: map[int64][]mediasvc.PhotoRecord{},
	}, candidateMediaStorageStub{})
	handler := NewCandidateHandler(service, mediaService, nil)

	cases := []struct {
		name string
		id   string
	}{
		{name: "pending", id: "101"},
		{name: "rejected", id: "102"},
		{name: "blocked", id: "103"},
		{name: "banned", id: "104"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/candidates/"+tc.id+"/media/photos", nil)
			req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
				UserID: 10,
				SID:    "sid-10",
				Role:   "USER",
			}))
			req = req.WithContext(withURLParam(req.Context(), "user_id", tc.id))

			rr := httptest.NewRecorder()
			handler.Photos(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}
