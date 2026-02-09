package stats

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"bot_moderator/internal/domain/model"
)

type fixtureAction struct {
	ActorTGID int64
	ActorRole string
	Username  string
	CreatedAt time.Time
}

type fakeRepo struct {
	actions []fixtureAction
	bounds  PeriodBounds
}

func (r *fakeRepo) Aggregate(_ context.Context, bounds PeriodBounds) (model.WorkStatsReport, error) {
	r.bounds = bounds

	type actorKey struct {
		tgID int64
		role string
		user string
	}
	actorsMap := make(map[actorKey]*model.WorkStatsActor)
	totals := model.WorkStatsTotals{}

	inRange := func(value, start, end time.Time) bool {
		return !value.Before(start) && value.Before(end)
	}

	for _, action := range r.actions {
		totals.All++
		if inRange(action.CreatedAt, bounds.DayStart, bounds.DayEnd) {
			totals.Day++
		}
		if inRange(action.CreatedAt, bounds.WeekStart, bounds.WeekEnd) {
			totals.Week++
		}
		if inRange(action.CreatedAt, bounds.MonthStart, bounds.MonthEnd) {
			totals.Month++
		}

		key := actorKey{tgID: action.ActorTGID, role: action.ActorRole, user: action.Username}
		actor, ok := actorsMap[key]
		if !ok {
			actor = &model.WorkStatsActor{
				ActorTGID: action.ActorTGID,
				ActorRole: action.ActorRole,
				Username:  action.Username,
			}
			actorsMap[key] = actor
		}

		actor.All++
		if inRange(action.CreatedAt, bounds.DayStart, bounds.DayEnd) {
			actor.Day++
		}
		if inRange(action.CreatedAt, bounds.WeekStart, bounds.WeekEnd) {
			actor.Week++
		}
		if inRange(action.CreatedAt, bounds.MonthStart, bounds.MonthEnd) {
			actor.Month++
		}
	}

	actors := make([]model.WorkStatsActor, 0, len(actorsMap))
	for _, actor := range actorsMap {
		actors = append(actors, *actor)
	}
	sort.SliceStable(actors, func(i, j int) bool {
		if actors[i].All != actors[j].All {
			return actors[i].All > actors[j].All
		}
		if actors[i].ActorTGID != actors[j].ActorTGID {
			return actors[i].ActorTGID < actors[j].ActorTGID
		}
		return actors[i].ActorRole < actors[j].ActorRole
	})

	return model.WorkStatsReport{
		Totals: totals,
		Actors: actors,
	}, nil
}

func TestBuildReportEuropeMinskBoundaries(t *testing.T) {
	loc, err := time.LoadLocation(minskLocationName)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	nowLocal := time.Date(2026, 2, 18, 12, 30, 0, 0, loc)
	dayStart := time.Date(2026, 2, 18, 0, 0, 0, 0, loc)
	dayEnd := dayStart.AddDate(0, 0, 1)
	weekStart := time.Date(2026, 2, 16, 0, 0, 0, 0, loc)
	weekEnd := weekStart.AddDate(0, 0, 7)
	monthStart := time.Date(2026, 2, 1, 0, 0, 0, 0, loc)
	monthEnd := monthStart.AddDate(0, 1, 0)

	repo := &fakeRepo{
		actions: []fixtureAction{
			{ActorTGID: 1001, ActorRole: "MODERATOR", Username: "mod1", CreatedAt: dayStart},
			{ActorTGID: 1001, ActorRole: "MODERATOR", Username: "mod1", CreatedAt: dayEnd},
			{ActorTGID: 1001, ActorRole: "MODERATOR", Username: "mod1", CreatedAt: weekStart},
			{ActorTGID: 1001, ActorRole: "MODERATOR", Username: "mod1", CreatedAt: weekStart.Add(-time.Nanosecond)},
			{ActorTGID: 1002, ActorRole: "ADMIN", Username: "", CreatedAt: monthStart},
			{ActorTGID: 1002, ActorRole: "ADMIN", Username: "", CreatedAt: monthEnd},
			{ActorTGID: 1002, ActorRole: "ADMIN", Username: "", CreatedAt: monthStart.Add(-time.Nanosecond)},
		},
	}

	service := newService(repo, func() time.Time { return nowLocal.UTC() }, loc)
	report, err := service.BuildReport(context.Background())
	if err != nil {
		t.Fatalf("build report: %v", err)
	}

	if repo.bounds.DayStart.Location().String() != minskLocationName {
		t.Fatalf("expected day start location %q, got %q", minskLocationName, repo.bounds.DayStart.Location().String())
	}
	if !repo.bounds.DayStart.Equal(dayStart) || !repo.bounds.DayEnd.Equal(dayEnd) {
		t.Fatalf("unexpected day bounds: %#v", repo.bounds)
	}
	if !repo.bounds.WeekStart.Equal(weekStart) || !repo.bounds.WeekEnd.Equal(weekEnd) {
		t.Fatalf("unexpected week bounds: %#v", repo.bounds)
	}
	if !repo.bounds.MonthStart.Equal(monthStart) || !repo.bounds.MonthEnd.Equal(monthEnd) {
		t.Fatalf("unexpected month bounds: %#v", repo.bounds)
	}

	if report.Totals.Day != 1 || report.Totals.Week != 3 || report.Totals.Month != 5 || report.Totals.All != 7 {
		t.Fatalf("unexpected totals: %+v", report.Totals)
	}

	actors := make(map[string]model.WorkStatsActor, len(report.Actors))
	for _, actor := range report.Actors {
		key := fmt.Sprintf("%d:%s", actor.ActorTGID, actor.ActorRole)
		actors[key] = actor
	}

	actorOne, ok := actors["1001:MODERATOR"]
	if !ok {
		t.Fatalf("missing stats for actor 1001 MODERATOR")
	}
	if actorOne.Day != 1 || actorOne.Week != 3 || actorOne.Month != 4 || actorOne.All != 4 {
		t.Fatalf("unexpected actorOne stats: %+v", actorOne)
	}

	actorTwo, ok := actors["1002:ADMIN"]
	if !ok {
		t.Fatalf("missing stats for actor 1002 ADMIN")
	}
	if actorTwo.Day != 0 || actorTwo.Week != 0 || actorTwo.Month != 1 || actorTwo.All != 3 {
		t.Fatalf("unexpected actorTwo stats: %+v", actorTwo)
	}
}
