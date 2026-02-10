package matches

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ivankudzin/tgapp/backend/internal/domain/enums"
	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

var (
	ErrValidation          = errors.New("validation error")
	ErrInvalidReportReason = errors.New("invalid report reason")
)

const (
	reportRateWindow = 10 * time.Minute
	reportRateRetry  = 10
)

type MatchStore interface {
	ListActiveForUser(ctx context.Context, userID int64, limit int) ([]pgrepo.ActiveMatchRecord, error)
	DeleteByUsers(ctx context.Context, tx pgx.Tx, userID, targetID int64) (bool, error)
}

type BlockStore interface {
	Upsert(ctx context.Context, tx pgx.Tx, actorUserID, targetUserID int64, reason string) error
}

type ReportStore interface {
	Create(
		ctx context.Context,
		tx pgx.Tx,
		reporterUserID, targetUserID int64,
		reason, details string,
		reporterTrustScore int,
		reporterRole string,
	) error
}

type ReportRateStore interface {
	IncrementWindow(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
}

type DailyMetricsStore interface {
	Increment(ctx context.Context, userID int64, at time.Time, delta pgrepo.DailyMetricsDelta) error
}

type TooManyReportsError struct {
	RetryAfterSec int64
}

func (e TooManyReportsError) Error() string {
	return "too many reports"
}

func (e TooManyReportsError) RetryAfter() int64 {
	if e.RetryAfterSec <= 0 {
		return 1
	}
	return e.RetryAfterSec
}

func IsTooManyReports(err error) (*TooManyReportsError, bool) {
	var rl TooManyReportsError
	if errors.As(err, &rl) {
		return &rl, true
	}
	return nil, false
}

type TempUnavailableError struct {
	RetryAfterSec int64
}

func (e TempUnavailableError) Error() string {
	return "temporarily unavailable"
}

func (e TempUnavailableError) RetryAfter() int64 {
	if e.RetryAfterSec <= 0 {
		return reportRateRetry
	}
	return e.RetryAfterSec
}

func IsTempUnavailable(err error) (*TempUnavailableError, bool) {
	var tu TempUnavailableError
	if errors.As(err, &tu) {
		return &tu, true
	}
	return nil, false
}

type Service struct {
	pool              *pgxpool.Pool
	matchStore        MatchStore
	blockStore        BlockStore
	reportStore       ReportStore
	reportRateStore   ReportRateStore
	reportMaxPer10Min int
	dailyMetrics      DailyMetricsStore
}

type Dependencies struct {
	Pool              *pgxpool.Pool
	MatchStore        MatchStore
	BlockStore        BlockStore
	ReportStore       ReportStore
	ReportRateStore   ReportRateStore
	ReportMaxPer10Min int
}

type MatchItem struct {
	ID           int64
	TargetUserID int64
	DisplayName  string
	Age          int
	CityID       string
	City         string
	CreatedAt    time.Time
}

func NewService(deps Dependencies) *Service {
	reportMaxPer10Min := deps.ReportMaxPer10Min
	if reportMaxPer10Min <= 0 {
		reportMaxPer10Min = 3
	}

	return &Service{
		pool:              deps.Pool,
		matchStore:        deps.MatchStore,
		blockStore:        deps.BlockStore,
		reportStore:       deps.ReportStore,
		reportRateStore:   deps.ReportRateStore,
		reportMaxPer10Min: reportMaxPer10Min,
	}
}

func (s *Service) AttachDailyMetrics(store DailyMetricsStore) {
	s.dailyMetrics = store
}

func (s *Service) List(ctx context.Context, userID int64, limit int) ([]MatchItem, error) {
	if userID <= 0 {
		return nil, ErrValidation
	}
	if s.matchStore == nil {
		return nil, fmt.Errorf("match store is nil")
	}

	rows, err := s.matchStore.ListActiveForUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	items := make([]MatchItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, MatchItem{
			ID:           row.ID,
			TargetUserID: row.TargetUserID,
			DisplayName:  row.DisplayName,
			Age:          row.Age,
			CityID:       row.CityID,
			City:         row.City,
			CreatedAt:    row.CreatedAt,
		})
	}
	return items, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, targetID int64) (bool, error) {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return false, ErrValidation
	}
	if s.pool == nil || s.matchStore == nil {
		return false, fmt.Errorf("unmatch dependencies are not configured")
	}

	var deleted bool
	if err := pgrepo.WithTx(ctx, s.pool, func(txCtx context.Context, tx pgx.Tx) error {
		ok, err := s.matchStore.DeleteByUsers(txCtx, tx, userID, targetID)
		if err != nil {
			return err
		}
		deleted = ok
		return nil
	}); err != nil {
		return false, err
	}

	return deleted, nil
}

func (s *Service) Block(ctx context.Context, userID, targetID int64, reason string) error {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return ErrValidation
	}
	if s.pool == nil || s.blockStore == nil || s.matchStore == nil {
		return fmt.Errorf("block dependencies are not configured")
	}

	return pgrepo.WithTx(ctx, s.pool, func(txCtx context.Context, tx pgx.Tx) error {
		if err := s.blockStore.Upsert(txCtx, tx, userID, targetID, reason); err != nil {
			return err
		}
		_, err := s.matchStore.DeleteByUsers(txCtx, tx, userID, targetID)
		return err
	})
}

func (s *Service) Report(ctx context.Context, userID, targetID int64, reason, details, reporterRole string) error {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return ErrValidation
	}
	if !isAllowedReason(reason) {
		return ErrInvalidReportReason
	}
	if err := s.checkReportRate(ctx, userID); err != nil {
		return err
	}
	if s.pool == nil || s.reportStore == nil {
		return fmt.Errorf("report dependencies are not configured")
	}

	normalizedReason := strings.ToLower(strings.TrimSpace(reason))
	normalizedRole := normalizeReporterRole(reporterRole)
	trustScore := reporterTrustScore(normalizedRole)

	if err := pgrepo.WithTx(ctx, s.pool, func(txCtx context.Context, tx pgx.Tx) error {
		return s.reportStore.Create(
			txCtx,
			tx,
			userID,
			targetID,
			normalizedReason,
			details,
			trustScore,
			normalizedRole,
		)
	}); err != nil {
		return err
	}
	if err := s.incrementReportMetric(ctx, userID); err != nil {
		log.Printf("warning: increment daily metrics failed for report: %v", err)
	}
	return nil
}

func isAllowedReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case string(enums.ReportReasonSpam),
		string(enums.ReportReasonFake),
		string(enums.ReportReasonAbusive),
		string(enums.ReportReasonOther):
		return true
	default:
		return false
	}
}

func (s *Service) checkReportRate(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return ErrValidation
	}
	if s.reportRateStore == nil || s.reportMaxPer10Min <= 0 {
		return nil
	}

	count, ttl, err := s.reportRateStore.IncrementWindow(ctx, reportRateKey(userID), reportRateWindow)
	if err != nil {
		log.Printf("warning: report rate limiter redis unavailable: %v", err)
		return TempUnavailableError{RetryAfterSec: reportRateRetry}
	}
	if count <= int64(s.reportMaxPer10Min) {
		return nil
	}

	return TooManyReportsError{
		RetryAfterSec: ceilSeconds(ttl),
	}
}

func reportRateKey(userID int64) string {
	return "rl:report:user:" + strconv.FormatInt(userID, 10) + ":10m"
}

func ceilSeconds(ttl time.Duration) int64 {
	if ttl <= 0 {
		return 1
	}
	sec := ttl / time.Second
	if ttl%time.Second != 0 {
		sec++
	}
	if sec < 1 {
		return 1
	}
	return int64(sec)
}

func normalizeReporterRole(role string) string {
	value := strings.ToLower(strings.TrimSpace(role))
	switch value {
	case string(enums.RoleAdmin), string(enums.RoleModerator), string(enums.RoleUser):
		return value
	default:
		return string(enums.RoleUser)
	}
}

func reporterTrustScore(role string) int {
	switch normalizeReporterRole(role) {
	case string(enums.RoleAdmin):
		return 100
	case string(enums.RoleModerator):
		return 50
	default:
		return 10
	}
}

func (s *Service) incrementReportMetric(ctx context.Context, userID int64) error {
	if s.dailyMetrics == nil || userID <= 0 {
		return nil
	}
	return s.dailyMetrics.Increment(ctx, userID, time.Now().UTC(), pgrepo.DailyMetricsDelta{
		Reports: 1,
	})
}
