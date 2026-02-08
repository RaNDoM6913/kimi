package feed

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

const (
	defaultPageSize = 20
	maxPageSize     = 50
)

var (
	ErrValidation    = errors.New("validation error")
	ErrInvalidCursor = errors.New("invalid cursor")
)

type Repository interface {
	GetViewerContext(ctx context.Context, userID int64) (pgrepo.FeedViewerContext, error)
	ListCandidates(ctx context.Context, q pgrepo.FeedQuery) ([]pgrepo.FeedCandidate, error)
}

type AdStore interface {
	ListActive(ctx context.Context, cityID string, limit int, at time.Time) ([]pgrepo.AdCardRecord, error)
}

type PlusStore interface {
	IsPlusActive(ctx context.Context, userID int64, at time.Time) (bool, *time.Time, error)
}

type Config struct {
	DefaultAgeMin   int
	DefaultAgeMax   int
	DefaultRadiusKM int
	MaxRadiusKM     int
}

type AdsConfig struct {
	FreeEvery     int
	PlusEvery     int
	DefaultIsPlus bool
}

type Service struct {
	repo      Repository
	cfg       Config
	adStore   AdStore
	plusStore PlusStore
	adsCfg    AdsConfig
	now       func() time.Time
}

type AdCard struct {
	ID       int64
	Kind     string
	Title    string
	AssetURL string
	ClickURL string
}

type Item struct {
	IsAd        bool
	Ad          *AdCard
	UserID      int64
	DisplayName string
	CityID      string
	City        string
	Age         int
	DistanceKM  *float64
}

type Result struct {
	Items      []Item
	NextCursor string
}

type pageCursor struct {
	Priority  int   `json:"p"`
	CreatedAt int64 `json:"t"`
	UserID    int64 `json:"i"`
}

func NewService(repo Repository, cfg Config) *Service {
	if cfg.DefaultAgeMin <= 0 {
		cfg.DefaultAgeMin = 18
	}
	if cfg.DefaultAgeMax <= 0 {
		cfg.DefaultAgeMax = 30
	}
	if cfg.DefaultRadiusKM <= 0 {
		cfg.DefaultRadiusKM = 3
	}
	if cfg.MaxRadiusKM <= 0 {
		cfg.MaxRadiusKM = 50
	}

	return &Service{
		repo: repo,
		cfg:  cfg,
		now:  time.Now,
	}
}

func (s *Service) AttachAds(adStore AdStore, plusStore PlusStore, cfg AdsConfig) {
	if cfg.FreeEvery <= 0 {
		cfg.FreeEvery = 7
	}
	if cfg.PlusEvery <= 0 {
		cfg.PlusEvery = 37
	}

	s.adStore = adStore
	s.plusStore = plusStore
	s.adsCfg = cfg
}

func (s *Service) Get(ctx context.Context, userID int64, cursor string, limit int) (Result, error) {
	if userID <= 0 {
		return Result{}, ErrValidation
	}
	if s.repo == nil {
		return Result{}, fmt.Errorf("feed repository is nil")
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	decoded, hasCursor, err := decodeCursor(cursor)
	if err != nil {
		return Result{}, err
	}

	viewer, err := s.repo.GetViewerContext(ctx, userID)
	if err != nil {
		if errors.Is(err, pgrepo.ErrFeedViewerNotFound) {
			return Result{Items: []Item{}}, nil
		}
		return Result{}, err
	}

	if strings.TrimSpace(viewer.CityID) == "" {
		return Result{Items: []Item{}}, nil
	}

	ageMin, ageMax := normalizeAgeRange(viewer.AgeMin, viewer.AgeMax, s.cfg.DefaultAgeMin, s.cfg.DefaultAgeMax)
	radius := normalizeRadius(viewer.RadiusKM, s.cfg.DefaultRadiusKM, s.cfg.MaxRadiusKM)
	query := pgrepo.FeedQuery{
		ViewerUserID:     userID,
		ViewerCityID:     viewer.CityID,
		ViewerGender:     viewer.Gender,
		ViewerLookingFor: viewer.LookingFor,
		ViewerGoals:      viewer.Goals,
		AgeMin:           ageMin,
		AgeMax:           ageMax,
		RadiusKM:         radius,
		ViewerLat:        viewer.LastLat,
		ViewerLon:        viewer.LastLon,
		HasCursor:        hasCursor,
		Limit:            limit,
		Now:              s.now().UTC(),
	}
	if hasCursor {
		query.CursorPriority = decoded.Priority
		query.CursorCreatedAt = time.UnixMilli(decoded.CreatedAt).UTC()
		query.CursorUserID = decoded.UserID
	}

	candidates, err := s.repo.ListCandidates(ctx, query)
	if err != nil {
		return Result{}, err
	}

	items := make([]Item, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, Item{
			UserID:      candidate.UserID,
			DisplayName: candidate.DisplayName,
			CityID:      candidate.CityID,
			City:        candidate.City,
			Age:         candidate.Age,
			DistanceKM:  candidate.DistanceKM,
		})
	}

	items = s.injectAds(ctx, userID, viewer.CityID, items)

	result := Result{Items: items}
	if len(candidates) == limit {
		last := candidates[len(candidates)-1]
		next, err := encodeCursor(pageCursor{
			Priority:  last.GoalsPriority,
			CreatedAt: last.CreatedAt.UTC().UnixMilli(),
			UserID:    last.UserID,
		})
		if err != nil {
			return Result{}, err
		}
		result.NextCursor = next
	}

	return result, nil
}

func (s *Service) injectAds(ctx context.Context, userID int64, cityID string, items []Item) []Item {
	if len(items) == 0 || s.adStore == nil || userID <= 0 {
		return items
	}

	every, ok := s.resolveAdsInterval(ctx, userID, s.now().UTC())
	if !ok || every <= 0 {
		return items
	}

	slots := len(items) / every
	if slots <= 0 {
		return items
	}

	ads, err := s.adStore.ListActive(ctx, cityID, slots, s.now().UTC())
	if err != nil || len(ads) == 0 {
		return items
	}

	out := make([]Item, 0, len(items)+slots)
	profilesSeen := 0
	insertions := 0
	for _, item := range items {
		out = append(out, item)
		profilesSeen++
		if profilesSeen%every != 0 {
			continue
		}

		ad := ads[insertions%len(ads)]
		insertions++
		out = append(out, Item{
			IsAd: true,
			Ad: &AdCard{
				ID:       ad.ID,
				Kind:     strings.ToUpper(strings.TrimSpace(ad.Kind)),
				Title:    ad.Title,
				AssetURL: ad.AssetURL,
				ClickURL: ad.ClickURL,
			},
		})
	}

	return out
}

func (s *Service) resolveAdsInterval(ctx context.Context, userID int64, at time.Time) (int, bool) {
	isPlus, err := s.resolvePlus(ctx, userID, at)
	if err != nil {
		return 0, false
	}

	if isPlus {
		return s.adsCfg.PlusEvery, true
	}
	return s.adsCfg.FreeEvery, true
}

func (s *Service) resolvePlus(ctx context.Context, userID int64, at time.Time) (bool, error) {
	if s.plusStore == nil {
		return s.adsCfg.DefaultIsPlus, nil
	}

	isPlus, _, err := s.plusStore.IsPlusActive(ctx, userID, at)
	if err != nil {
		return false, fmt.Errorf("resolve plus for ads: %w", err)
	}
	if isPlus {
		return true, nil
	}

	return s.adsCfg.DefaultIsPlus, nil
}

func normalizeAgeRange(ageMin, ageMax, defaultMin, defaultMax int) (int, int) {
	if ageMin <= 0 {
		ageMin = defaultMin
	}
	if ageMax <= 0 {
		ageMax = defaultMax
	}
	if ageMin > ageMax {
		ageMin, ageMax = ageMax, ageMin
	}
	return ageMin, ageMax
}

func normalizeRadius(radius, fallback, max int) int {
	if radius <= 0 {
		radius = fallback
	}
	if radius > max {
		radius = max
	}
	return radius
}

func decodeCursor(raw string) (pageCursor, bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return pageCursor{}, false, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return pageCursor{}, false, ErrInvalidCursor
	}

	var cursor pageCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return pageCursor{}, false, ErrInvalidCursor
	}
	if cursor.CreatedAt <= 0 || cursor.UserID <= 0 || cursor.Priority < 0 || cursor.Priority > 1 {
		return pageCursor{}, false, ErrInvalidCursor
	}

	return cursor, true, nil
}

func encodeCursor(cursor pageCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal feed cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}
