package feed

import (
	"context"
	"errors"
	"testing"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	antiabusesvc "github.com/ivankudzin/tgapp/backend/internal/services/antiabuse"
)

type feedRepoStub struct {
	viewer    pgrepo.FeedViewerContext
	viewerErr error
	items     []pgrepo.FeedCandidate
	lastQuery pgrepo.FeedQuery
	candidate pgrepo.CandidateProfileRecord
	candErr   error
}

func float64ptr(v float64) *float64 {
	return &v
}

type feedAdStoreStub struct {
	items []pgrepo.AdCardRecord
}

func (s *feedAdStoreStub) ListActive(_ context.Context, _ string, limit int, _ time.Time) ([]pgrepo.AdCardRecord, error) {
	if limit <= 0 || limit > len(s.items) {
		limit = len(s.items)
	}
	out := make([]pgrepo.AdCardRecord, 0, limit)
	out = append(out, s.items[:limit]...)
	return out, nil
}

type feedPlusStoreStub struct {
	isPlus bool
}

type feedAntiAbuseStub struct {
	shadow map[int64]bool
}

func (s *feedPlusStoreStub) IsPlusActive(_ context.Context, _ int64, _ time.Time) (bool, *time.Time, error) {
	return s.isPlus, nil, nil
}

func (s *feedAntiAbuseStub) GetState(_ context.Context, userID int64) (antiabusesvc.State, error) {
	if s.shadow != nil && s.shadow[userID] {
		return antiabusesvc.State{RiskScore: 5, ShadowEnabled: true, Exists: true}, nil
	}
	return antiabusesvc.State{RiskScore: 0, ShadowEnabled: false, Exists: true}, nil
}

func (s *feedRepoStub) GetViewerContext(_ context.Context, _ int64) (pgrepo.FeedViewerContext, error) {
	if s.viewerErr != nil {
		return pgrepo.FeedViewerContext{}, s.viewerErr
	}
	return s.viewer, nil
}

func (s *feedRepoStub) ListCandidates(_ context.Context, q pgrepo.FeedQuery) ([]pgrepo.FeedCandidate, error) {
	s.lastQuery = q
	limit := q.Limit
	if limit <= 0 || limit > len(s.items) {
		limit = len(s.items)
	}
	out := make([]pgrepo.FeedCandidate, 0, limit)
	out = append(out, s.items[:limit]...)
	return out, nil
}

func (s *feedRepoStub) GetCandidateProfile(_ context.Context, _ pgrepo.CandidateProfileQuery) (pgrepo.CandidateProfileRecord, error) {
	if s.candErr != nil {
		return pgrepo.CandidateProfileRecord{}, s.candErr
	}
	if s.candidate.UserID <= 0 {
		return pgrepo.CandidateProfileRecord{}, pgrepo.ErrFeedCandidateNotFound
	}
	return s.candidate, nil
}

func TestGetUsesDefaultsAndReturnsNextCursor(t *testing.T) {
	repo := &feedRepoStub{
		viewer: pgrepo.FeedViewerContext{
			UserID:     10,
			CityID:     "minsk",
			Gender:     "male",
			LookingFor: "female",
			AgeMin:     0,
			AgeMax:     0,
			RadiusKM:   0,
			Goals:      []string{"relationship"},
		},
		items: []pgrepo.FeedCandidate{
			{
				UserID:        200,
				DisplayName:   "Anna",
				CityID:        "minsk",
				City:          "Minsk",
				Age:           25,
				GoalsPriority: 1,
				CreatedAt:     time.Date(2026, 2, 8, 11, 0, 0, 0, time.UTC),
			},
			{
				UserID:        199,
				DisplayName:   "Kate",
				CityID:        "minsk",
				City:          "Minsk",
				Age:           24,
				GoalsPriority: 0,
				CreatedAt:     time.Date(2026, 2, 8, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	service := NewService(repo, Config{
		DefaultAgeMin:   18,
		DefaultAgeMax:   30,
		DefaultRadiusKM: 3,
		MaxRadiusKM:     50,
	})

	result, err := service.Get(context.Background(), 10, "", 2)
	if err != nil {
		t.Fatalf("get feed: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("unexpected items count: got %d want %d", len(result.Items), 2)
	}
	if result.NextCursor == "" {
		t.Fatalf("expected next cursor for full page")
	}

	if repo.lastQuery.AgeMin != 18 || repo.lastQuery.AgeMax != 30 {
		t.Fatalf("unexpected age filter defaults: got %d-%d", repo.lastQuery.AgeMin, repo.lastQuery.AgeMax)
	}
	if repo.lastQuery.RadiusKM != 3 {
		t.Fatalf("unexpected radius default: got %d want %d", repo.lastQuery.RadiusKM, 3)
	}

	_, err = service.Get(context.Background(), 10, result.NextCursor, 2)
	if err != nil {
		t.Fatalf("get feed with cursor: %v", err)
	}
	if !repo.lastQuery.HasCursor {
		t.Fatalf("expected cursor query flag")
	}
	if repo.lastQuery.CursorUserID != 199 {
		t.Fatalf("unexpected cursor user id: got %d want %d", repo.lastQuery.CursorUserID, 199)
	}
}

func TestGetInvalidCursor(t *testing.T) {
	repo := &feedRepoStub{
		viewer: pgrepo.FeedViewerContext{
			UserID: 1,
			CityID: "minsk",
		},
	}

	service := NewService(repo, Config{})
	_, err := service.Get(context.Background(), 1, "%%%invalid%%%", 20)
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestGetInjectsAdsByInterval(t *testing.T) {
	repo := &feedRepoStub{
		viewer: pgrepo.FeedViewerContext{
			UserID:     10,
			CityID:     "minsk",
			Gender:     "male",
			LookingFor: "female",
			AgeMin:     18,
			AgeMax:     30,
			RadiusKM:   3,
			Goals:      []string{"relationship"},
		},
		items: []pgrepo.FeedCandidate{
			{UserID: 1, DisplayName: "u1", CityID: "minsk", City: "Minsk", Age: 21, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 12, 0, 0, 0, time.UTC)},
			{UserID: 2, DisplayName: "u2", CityID: "minsk", City: "Minsk", Age: 22, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 59, 0, 0, time.UTC)},
			{UserID: 3, DisplayName: "u3", CityID: "minsk", City: "Minsk", Age: 23, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 58, 0, 0, time.UTC)},
			{UserID: 4, DisplayName: "u4", CityID: "minsk", City: "Minsk", Age: 24, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 57, 0, 0, time.UTC)},
			{UserID: 5, DisplayName: "u5", CityID: "minsk", City: "Minsk", Age: 25, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 56, 0, 0, time.UTC)},
			{UserID: 6, DisplayName: "u6", CityID: "minsk", City: "Minsk", Age: 26, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 55, 0, 0, time.UTC)},
			{UserID: 7, DisplayName: "u7", CityID: "minsk", City: "Minsk", Age: 27, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 54, 0, 0, time.UTC)},
			{UserID: 8, DisplayName: "u8", CityID: "minsk", City: "Minsk", Age: 28, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 53, 0, 0, time.UTC)},
		},
	}

	service := NewService(repo, Config{
		DefaultAgeMin:   18,
		DefaultAgeMax:   30,
		DefaultRadiusKM: 3,
		MaxRadiusKM:     50,
	})
	service.AttachAds(
		&feedAdStoreStub{
			items: []pgrepo.AdCardRecord{
				{ID: 101, Kind: "IMAGE", Title: "Ad1", AssetURL: "s3://ad1", ClickURL: "https://ad1"},
			},
		},
		&feedPlusStoreStub{isPlus: false},
		AdsConfig{
			FreeEvery: 3,
			PlusEvery: 37,
		},
	)

	result, err := service.Get(context.Background(), 10, "", 8)
	if err != nil {
		t.Fatalf("get feed with ads: %v", err)
	}
	if len(result.Items) != 10 {
		t.Fatalf("unexpected mixed items count: got %d want %d", len(result.Items), 10)
	}
	if !result.Items[3].IsAd || result.Items[3].Ad == nil || result.Items[3].Ad.ID != 101 {
		t.Fatalf("expected ad at index 3")
	}
	if !result.Items[7].IsAd || result.Items[7].Ad == nil || result.Items[7].Ad.ID != 101 {
		t.Fatalf("expected ad at index 7")
	}
}

func TestGetAppliesShadowRankMultiplierAndKeepsCursorOrder(t *testing.T) {
	repo := &feedRepoStub{
		viewer: pgrepo.FeedViewerContext{
			UserID:     10,
			CityID:     "minsk",
			Gender:     "male",
			LookingFor: "female",
			AgeMin:     18,
			AgeMax:     30,
			RadiusKM:   3,
			Goals:      []string{"relationship"},
		},
		items: []pgrepo.FeedCandidate{
			{
				UserID:        301,
				DisplayName:   "shadow-first",
				CityID:        "minsk",
				City:          "Minsk",
				Age:           25,
				GoalsPriority: 1,
				RankScore:     float64ptr(1.0),
				CreatedAt:     time.Date(2026, 2, 8, 12, 0, 0, 0, time.UTC),
			},
			{
				UserID:        300,
				DisplayName:   "normal-second",
				CityID:        "minsk",
				City:          "Minsk",
				Age:           24,
				GoalsPriority: 1,
				RankScore:     float64ptr(0.7),
				CreatedAt:     time.Date(2026, 2, 8, 11, 59, 0, 0, time.UTC),
			},
		},
	}

	service := NewService(repo, Config{
		DefaultAgeMin:   18,
		DefaultAgeMax:   30,
		DefaultRadiusKM: 3,
		MaxRadiusKM:     50,
	})
	service.AttachAntiAbuse(&feedAntiAbuseStub{
		shadow: map[int64]bool{
			301: true,
		},
	}, 0.4)

	result, err := service.Get(context.Background(), 10, "", 2)
	if err != nil {
		t.Fatalf("get feed with shadow demotion: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("unexpected items count: got %d want %d", len(result.Items), 2)
	}
	if result.Items[0].UserID != 300 || result.Items[1].UserID != 301 {
		t.Fatalf("unexpected order after multiplier demotion: got [%d,%d] want [300,301]", result.Items[0].UserID, result.Items[1].UserID)
	}
	if result.NextCursor == "" {
		t.Fatalf("expected next cursor")
	}

	if _, err := service.Get(context.Background(), 10, result.NextCursor, 2); err != nil {
		t.Fatalf("get feed with demoted cursor: %v", err)
	}
	if repo.lastQuery.CursorUserID != 300 {
		t.Fatalf("cursor must follow base SQL order, got %d want %d", repo.lastQuery.CursorUserID, 300)
	}
}

func TestGetDilutesShadowCandidatesWhenScoreIsMissing(t *testing.T) {
	repo := &feedRepoStub{
		viewer: pgrepo.FeedViewerContext{
			UserID:     10,
			CityID:     "minsk",
			Gender:     "male",
			LookingFor: "female",
			AgeMin:     18,
			AgeMax:     30,
			RadiusKM:   3,
			Goals:      []string{"relationship"},
		},
		items: []pgrepo.FeedCandidate{
			{UserID: 901, DisplayName: "shadow-1", CityID: "minsk", City: "Minsk", Age: 23, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 12, 0, 0, 0, time.UTC)},
			{UserID: 800, DisplayName: "normal-1", CityID: "minsk", City: "Minsk", Age: 22, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 59, 0, 0, time.UTC)},
			{UserID: 801, DisplayName: "normal-2", CityID: "minsk", City: "Minsk", Age: 22, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 58, 0, 0, time.UTC)},
			{UserID: 802, DisplayName: "normal-3", CityID: "minsk", City: "Minsk", Age: 22, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 57, 0, 0, time.UTC)},
			{UserID: 803, DisplayName: "normal-4", CityID: "minsk", City: "Minsk", Age: 22, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 56, 0, 0, time.UTC)},
			{UserID: 804, DisplayName: "normal-5", CityID: "minsk", City: "Minsk", Age: 22, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 55, 0, 0, time.UTC)},
			{UserID: 902, DisplayName: "shadow-2", CityID: "minsk", City: "Minsk", Age: 23, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 54, 0, 0, time.UTC)},
			{UserID: 805, DisplayName: "normal-6", CityID: "minsk", City: "Minsk", Age: 22, GoalsPriority: 1, CreatedAt: time.Date(2026, 2, 8, 11, 53, 0, 0, time.UTC)},
		},
	}

	service := NewService(repo, Config{
		DefaultAgeMin:   18,
		DefaultAgeMax:   30,
		DefaultRadiusKM: 3,
		MaxRadiusKM:     50,
	})
	service.AttachAntiAbuse(&feedAntiAbuseStub{
		shadow: map[int64]bool{
			901: true,
			902: true,
		},
	}, 0.4)

	result, err := service.Get(context.Background(), 10, "", 8)
	if err != nil {
		t.Fatalf("get feed with shadow dilution: %v", err)
	}
	if len(result.Items) != 8 {
		t.Fatalf("unexpected items count: got %d want %d", len(result.Items), 8)
	}

	wantOrder := []int64{800, 801, 802, 803, 804, 901, 805, 902}
	for i, userID := range wantOrder {
		if result.Items[i].UserID != userID {
			t.Fatalf("unexpected dilution order at index %d: got %d want %d", i, result.Items[i].UserID, userID)
		}
	}
}

func TestGetCandidateProfileReturnsMappedData(t *testing.T) {
	repo := &feedRepoStub{
		candidate: pgrepo.CandidateProfileRecord{
			UserID:      55,
			DisplayName: "Alice",
			Age:         25,
			Zodiac:      "aries",
			CityID:      "minsk",
			City:        "Minsk",
			Occupation:  "Designer",
			Education:   "BSU",
			HeightCM:    172,
			EyeColor:    "brown",
			Languages:   []string{"ru", "en"},
			Goals:       []string{"relationship"},
			IsPlus:      true,
		},
	}
	service := NewService(repo, Config{})

	got, err := service.GetCandidateProfile(context.Background(), 10, 55)
	if err != nil {
		t.Fatalf("get candidate profile: %v", err)
	}
	if got.UserID != 55 || got.Zodiac != "aries" {
		t.Fatalf("unexpected candidate profile payload: %+v", got)
	}
	if !got.Badges.IsPlus {
		t.Fatalf("expected plus badge to be true")
	}
}

func TestGetCandidateProfileNotFound(t *testing.T) {
	repo := &feedRepoStub{candErr: pgrepo.ErrFeedCandidateNotFound}
	service := NewService(repo, Config{})

	_, err := service.GetCandidateProfile(context.Background(), 10, 77)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
