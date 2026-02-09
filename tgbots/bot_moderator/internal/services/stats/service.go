package stats

import (
	"context"
	"time"

	"bot_moderator/internal/domain/model"
)

const minskLocationName = "Europe/Minsk"

type PeriodBounds struct {
	DayStart   time.Time
	DayEnd     time.Time
	WeekStart  time.Time
	WeekEnd    time.Time
	MonthStart time.Time
	MonthEnd   time.Time
}

type Repo interface {
	Aggregate(context.Context, PeriodBounds) (model.WorkStatsReport, error)
}

type Service struct {
	repo  Repo
	nowFn func() time.Time
	loc   *time.Location
}

func NewService(repo Repo) *Service {
	return newService(repo, time.Now, loadMinskLocation())
}

func newService(repo Repo, nowFn func() time.Time, loc *time.Location) *Service {
	if nowFn == nil {
		nowFn = time.Now
	}
	if loc == nil {
		loc = loadMinskLocation()
	}
	return &Service{
		repo:  repo,
		nowFn: nowFn,
		loc:   loc,
	}
}

func (s *Service) BuildReport(ctx context.Context) (model.WorkStatsReport, error) {
	if s.repo == nil {
		return model.WorkStatsReport{}, nil
	}

	bounds := computePeriodBounds(s.nowFn(), s.loc)
	return s.repo.Aggregate(ctx, bounds)
}

func computePeriodBounds(now time.Time, loc *time.Location) PeriodBounds {
	localNow := now.In(loc)
	year, month, day := localNow.Date()
	dayStart := time.Date(year, month, day, 0, 0, 0, 0, loc)
	dayEnd := dayStart.AddDate(0, 0, 1)

	weekday := int(dayStart.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := dayStart.AddDate(0, 0, -(weekday - 1))
	weekEnd := weekStart.AddDate(0, 0, 7)

	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	monthEnd := monthStart.AddDate(0, 1, 0)

	return PeriodBounds{
		DayStart:   dayStart,
		DayEnd:     dayEnd,
		WeekStart:  weekStart,
		WeekEnd:    weekEnd,
		MonthStart: monthStart,
		MonthEnd:   monthEnd,
	}
}

func loadMinskLocation() *time.Location {
	loc, err := time.LoadLocation(minskLocationName)
	if err != nil {
		return time.FixedZone(minskLocationName, 3*3600)
	}
	return loc
}
