package adminhttp

import (
	"context"

	"bot_moderator/internal/domain/model"
	"bot_moderator/internal/repo/postgres"
	statssvc "bot_moderator/internal/services/stats"
)

type WorkStatsRepo struct {
	client *Client
	db     *postgres.WorkStatsRepo
	dual   bool
}

func NewWorkStatsRepo(client *Client, db *postgres.WorkStatsRepo, dual bool) *WorkStatsRepo {
	return &WorkStatsRepo{
		client: client,
		db:     db,
		dual:   dual,
	}
}

func (r *WorkStatsRepo) Aggregate(ctx context.Context, bounds statssvc.PeriodBounds) (model.WorkStatsReport, error) {
	request := map[string]interface{}{
		"day_start":   bounds.DayStart,
		"day_end":     bounds.DayEnd,
		"week_start":  bounds.WeekStart,
		"week_end":    bounds.WeekEnd,
		"month_start": bounds.MonthStart,
		"month_end":   bounds.MonthEnd,
	}

	response := model.WorkStatsReport{}
	err := r.client.DoJSON(ctx, "POST", "/admin/bot/stats/work", request, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.Aggregate(ctx, bounds)
	}
	if err != nil {
		return model.WorkStatsReport{}, err
	}
	return response, nil
}
