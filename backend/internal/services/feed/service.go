package feed

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/domain/rules"
	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	antiabusesvc "github.com/ivankudzin/tgapp/backend/internal/services/antiabuse"
)

const (
	defaultPageSize = 20
	maxPageSize     = 50
	feedPhotoURLTTL = 5 * time.Minute
)

var (
	ErrValidation    = errors.New("validation error")
	ErrInvalidCursor = errors.New("invalid cursor")
	ErrNotFound      = errors.New("not found")
)

type Repository interface {
	GetViewerContext(ctx context.Context, userID int64) (pgrepo.FeedViewerContext, error)
	ListCandidates(ctx context.Context, q pgrepo.FeedQuery) ([]pgrepo.FeedCandidate, error)
	GetCandidateProfile(ctx context.Context, q pgrepo.CandidateProfileQuery) (pgrepo.CandidateProfileRecord, error)
}

type AdStore interface {
	ListActive(ctx context.Context, cityID string, limit int, at time.Time) ([]pgrepo.AdCardRecord, error)
}

type PlusStore interface {
	IsPlusActive(ctx context.Context, userID int64, at time.Time) (bool, *time.Time, error)
}

type PhotoURLSigner interface {
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type AntiAbuseStore interface {
	GetState(ctx context.Context, userID int64) (antiabusesvc.State, error)
}

type Config struct {
	DefaultAgeMin        int
	DefaultAgeMax        int
	DefaultRadiusKM      int
	MaxRadiusKM          int
	ShadowRankMultiplier float64
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
	photoSign PhotoURLSigner
	antiAbuse AntiAbuseStore
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
	IsAd            bool
	Ad              *AdCard
	UserID          int64
	DisplayName     string
	CityID          string
	City            string
	Zodiac          string
	PrimaryGoal     string
	PrimaryPhotoURL *string
	Age             int
	DistanceKM      *float64
}

type Result struct {
	Items      []Item
	NextCursor string
}

type CandidateBadges struct {
	IsPlus bool
}

type CandidateProfile struct {
	UserID      int64
	DisplayName string
	Age         int
	Zodiac      string
	CityID      string
	City        string
	DistanceKM  *float64
	Bio         *string
	Occupation  string
	Education   string
	HeightCM    int
	EyeColor    string
	Languages   []string
	Goals       []string
	IsTravel    bool
	TravelCity  *string
	Badges      CandidateBadges
}

type pageCursor struct {
	Priority  int   `json:"p"`
	CreatedAt int64 `json:"t"`
	UserID    int64 `json:"i"`
}

type candidateRank struct {
	candidate pgrepo.FeedCandidate
	shadow    bool
	score     float64
	hasScore  bool
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
	if cfg.ShadowRankMultiplier <= 0 || cfg.ShadowRankMultiplier > 1 {
		cfg.ShadowRankMultiplier = 0.4
	}

	return &Service{
		repo: repo,
		cfg:  cfg,
		now:  time.Now,
	}
}

func (s *Service) AttachAntiAbuse(antiAbuse AntiAbuseStore, shadowRankMultiplier float64) {
	s.antiAbuse = antiAbuse
	if shadowRankMultiplier > 0 && shadowRankMultiplier <= 1 {
		s.cfg.ShadowRankMultiplier = shadowRankMultiplier
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

func (s *Service) AttachPhotoSigner(signer PhotoURLSigner) {
	s.photoSign = signer
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
	cursorCandidates := candidates
	candidates = s.demoteShadowCandidates(ctx, candidates)

	items := make([]Item, 0, len(candidates))
	for _, candidate := range candidates {
		zodiac := strings.TrimSpace(candidate.Zodiac)
		if zodiac == "" && candidate.Birthdate != nil {
			zodiac = rules.ZodiacFromBirthdate(candidate.Birthdate.UTC())
		}

		items = append(items, Item{
			UserID:          candidate.UserID,
			DisplayName:     candidate.DisplayName,
			CityID:          candidate.CityID,
			City:            candidate.City,
			Zodiac:          zodiac,
			PrimaryGoal:     strings.TrimSpace(candidate.PrimaryGoal),
			PrimaryPhotoURL: s.buildPhotoURL(ctx, candidate.PrimaryPhoto),
			Age:             candidate.Age,
			DistanceKM:      candidate.DistanceKM,
		})
	}

	items = s.injectAds(ctx, userID, viewer.CityID, items)

	result := Result{Items: items}
	if len(cursorCandidates) == limit {
		last := cursorCandidates[len(cursorCandidates)-1]
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

func (s *Service) GetCandidateProfile(ctx context.Context, viewerUserID, candidateUserID int64) (CandidateProfile, error) {
	if viewerUserID <= 0 || candidateUserID <= 0 || viewerUserID == candidateUserID {
		return CandidateProfile{}, ErrValidation
	}
	if s.repo == nil {
		return CandidateProfile{}, fmt.Errorf("feed repository is nil")
	}

	record, err := s.repo.GetCandidateProfile(ctx, pgrepo.CandidateProfileQuery{
		ViewerUserID:    viewerUserID,
		CandidateUserID: candidateUserID,
		Now:             s.now().UTC(),
	})
	if err != nil {
		if errors.Is(err, pgrepo.ErrFeedCandidateNotFound) {
			return CandidateProfile{}, ErrNotFound
		}
		return CandidateProfile{}, err
	}

	return CandidateProfile{
		UserID:      record.UserID,
		DisplayName: record.DisplayName,
		Age:         record.Age,
		Zodiac:      record.Zodiac,
		CityID:      record.CityID,
		City:        record.City,
		DistanceKM:  record.DistanceKM,
		Bio:         record.Bio,
		Occupation:  record.Occupation,
		Education:   record.Education,
		HeightCM:    record.HeightCM,
		EyeColor:    record.EyeColor,
		Languages:   append([]string(nil), record.Languages...),
		Goals:       append([]string(nil), record.Goals...),
		IsTravel:    false,
		TravelCity:  nil,
		Badges: CandidateBadges{
			IsPlus: record.IsPlus,
		},
	}, nil
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

func (s *Service) buildPhotoURL(ctx context.Context, key string) *string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		value := trimmed
		return &value
	}
	if s.photoSign == nil {
		return nil
	}

	url, err := s.photoSign.PresignGet(ctx, trimmed, feedPhotoURLTTL)
	if err != nil {
		return nil
	}
	value := strings.TrimSpace(url)
	if value == "" {
		return nil
	}
	return &value
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

func (s *Service) demoteShadowCandidates(ctx context.Context, candidates []pgrepo.FeedCandidate) []pgrepo.FeedCandidate {
	if len(candidates) == 0 || s.antiAbuse == nil {
		return candidates
	}

	ranked := make([]candidateRank, 0, len(candidates))
	hasRankingScore := true
	for _, candidate := range candidates {
		state, err := s.antiAbuse.GetState(ctx, candidate.UserID)
		shadow := err == nil && state.ShadowEnabled
		if candidate.RankScore == nil {
			hasRankingScore = false
		}
		entry := candidateRank{
			candidate: candidate,
			shadow:    shadow,
			hasScore:  candidate.RankScore != nil,
		}
		if candidate.RankScore != nil {
			entry.score = *candidate.RankScore
			if shadow {
				entry.score = entry.score * s.cfg.ShadowRankMultiplier
			}
		}
		ranked = append(ranked, entry)
	}

	if hasRankingScore {
		sort.SliceStable(ranked, func(i, j int) bool {
			left := ranked[i]
			right := ranked[j]
			if left.score != right.score {
				return left.score > right.score
			}
			if !left.candidate.CreatedAt.Equal(right.candidate.CreatedAt) {
				return left.candidate.CreatedAt.After(right.candidate.CreatedAt)
			}
			return left.candidate.UserID > right.candidate.UserID
		})

		out := make([]pgrepo.FeedCandidate, 0, len(ranked))
		for _, item := range ranked {
			out = append(out, item.candidate)
		}
		return out
	}

	return diluteShadowCandidates(ranked, 5)
}

func diluteShadowCandidates(candidates []candidateRank, normalBatch int) []pgrepo.FeedCandidate {
	if normalBatch <= 0 {
		normalBatch = 5
	}

	normal := make([]pgrepo.FeedCandidate, 0, len(candidates))
	shadow := make([]pgrepo.FeedCandidate, 0, len(candidates))
	for _, item := range candidates {
		if item.shadow {
			shadow = append(shadow, item.candidate)
			continue
		}
		normal = append(normal, item.candidate)
	}

	out := make([]pgrepo.FeedCandidate, 0, len(candidates))
	normalPos := 0
	shadowPos := 0
	for normalPos < len(normal) || shadowPos < len(shadow) {
		batchEnd := normalPos + normalBatch
		if batchEnd > len(normal) {
			batchEnd = len(normal)
		}
		if normalPos < batchEnd {
			out = append(out, normal[normalPos:batchEnd]...)
			normalPos = batchEnd
		}
		if shadowPos < len(shadow) {
			out = append(out, shadow[shadowPos])
			shadowPos++
		}
		if normalPos >= len(normal) && shadowPos < len(shadow) {
			out = append(out, shadow[shadowPos:]...)
			break
		}
	}

	return out
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
