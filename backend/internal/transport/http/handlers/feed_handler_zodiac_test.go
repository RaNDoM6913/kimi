package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	feedsvc "github.com/ivankudzin/tgapp/backend/internal/services/feed"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
)

type feedRepoZodiacStub struct{}

func (feedRepoZodiacStub) GetViewerContext(context.Context, int64) (pgrepo.FeedViewerContext, error) {
	return pgrepo.FeedViewerContext{
		UserID:     10,
		CityID:     "minsk",
		Gender:     "male",
		LookingFor: "female",
		AgeMin:     18,
		AgeMax:     30,
		RadiusKM:   3,
		Goals:      []string{"relationship"},
	}, nil
}

func (feedRepoZodiacStub) ListCandidates(context.Context, pgrepo.FeedQuery) ([]pgrepo.FeedCandidate, error) {
	return []pgrepo.FeedCandidate{
		{
			UserID:       1001,
			DisplayName:  "Alice",
			CityID:       "minsk",
			City:         "Minsk",
			Zodiac:       "aquarius",
			PrimaryPhoto: "users/1001/photos/1.jpg",
			Age:          24,
			CreatedAt:    time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		},
	}, nil
}

func (feedRepoZodiacStub) GetCandidateProfile(context.Context, pgrepo.CandidateProfileQuery) (pgrepo.CandidateProfileRecord, error) {
	return pgrepo.CandidateProfileRecord{}, pgrepo.ErrFeedCandidateNotFound
}

type feedPhotoSignerStub struct{}

func (feedPhotoSignerStub) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://signed.local/" + key, nil
}

type feedAdStoreHandlerStub struct {
	items []pgrepo.AdCardRecord
}

func (s feedAdStoreHandlerStub) ListActive(_ context.Context, _ string, _ int, _ time.Time) ([]pgrepo.AdCardRecord, error) {
	return append([]pgrepo.AdCardRecord(nil), s.items...), nil
}

type feedPlusStoreHandlerStub struct {
	isPlus bool
}

func (s feedPlusStoreHandlerStub) IsPlusActive(_ context.Context, _ int64, _ time.Time) (bool, *time.Time, error) {
	return s.isPlus, nil, nil
}

func TestFeedReturnsZodiacInProfileItems(t *testing.T) {
	service := feedsvc.NewService(feedRepoZodiacStub{}, feedsvc.Config{})
	service.AttachPhotoSigner(feedPhotoSignerStub{})
	handler := NewFeedHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 10,
		SID:    "sid-10",
		Role:   "USER",
	}))
	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	var payload dto.FeedResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("unexpected items count: got %d want %d", len(payload.Items), 1)
	}
	if payload.Items[0].Zodiac != "aquarius" {
		t.Fatalf("unexpected zodiac in feed response: got %q want %q", payload.Items[0].Zodiac, "aquarius")
	}
	if payload.Items[0].PrimaryPhotoURL == nil ||
		payload.Items[0].PrimaryPhotoURL.Value == nil ||
		strings.TrimSpace(*payload.Items[0].PrimaryPhotoURL.Value) == "" {
		t.Fatalf("expected non-empty primary_photo_url")
	}
}

func TestFeedAdItemShapeUnchanged(t *testing.T) {
	service := feedsvc.NewService(feedRepoZodiacStub{}, feedsvc.Config{})
	service.AttachPhotoSigner(feedPhotoSignerStub{})
	service.AttachAds(
		feedAdStoreHandlerStub{
			items: []pgrepo.AdCardRecord{
				{
					ID:       500,
					Kind:     "IMAGE",
					Title:    "ad",
					AssetURL: "https://cdn.local/ad.jpg",
					ClickURL: "https://example.com/ad",
				},
			},
		},
		feedPlusStoreHandlerStub{isPlus: false},
		feedsvc.AdsConfig{
			FreeEvery: 1,
			PlusEvery: 37,
		},
	)
	handler := NewFeedHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/v1/feed?limit=1", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 10,
		SID:    "sid-10",
		Role:   "USER",
	}))
	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	rawItems, ok := payload["items"].([]any)
	if !ok || len(rawItems) != 2 {
		t.Fatalf("unexpected items payload: %#v", payload["items"])
	}
	adItem, ok := rawItems[1].(map[string]any)
	if !ok {
		t.Fatalf("unexpected ad item payload: %#v", rawItems[1])
	}
	if _, exists := adItem["ad"]; !exists {
		t.Fatalf("expected ad object in ad item")
	}
	if _, exists := adItem["display_name"]; exists {
		t.Fatalf("ad item must not contain profile field display_name")
	}
	if _, exists := adItem["primary_photo_url"]; exists {
		t.Fatalf("ad item must not contain profile field primary_photo_url")
	}
}

func TestFeedProfileWithoutPhotoReturnsNullPrimaryPhotoURL(t *testing.T) {
	service := feedsvc.NewService(feedRepoZodiacStub{}, feedsvc.Config{})
	handler := NewFeedHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/v1/feed?limit=1", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 10,
		SID:    "sid-10",
		Role:   "USER",
	}))
	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	rawItems, ok := payload["items"].([]any)
	if !ok || len(rawItems) == 0 {
		t.Fatalf("unexpected items payload: %#v", payload["items"])
	}
	item, ok := rawItems[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected first item payload: %#v", rawItems[0])
	}
	if _, exists := item["primary_photo_url"]; !exists {
		t.Fatalf("expected primary_photo_url field for profile item")
	}
	if item["primary_photo_url"] != nil {
		t.Fatalf("expected null primary_photo_url when no signer/photo, got %#v", item["primary_photo_url"])
	}
}
