package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"bot_moderator/internal/domain/model"
	statssvc "bot_moderator/internal/services/stats"
)

type WorkStatsRepo struct {
	db *sql.DB
}

func NewWorkStatsRepo(db *sql.DB) *WorkStatsRepo {
	return &WorkStatsRepo{db: db}
}

func (r *WorkStatsRepo) Aggregate(ctx context.Context, bounds statssvc.PeriodBounds) (model.WorkStatsReport, error) {
	if r.db == nil {
		return model.WorkStatsReport{}, nil
	}

	report := model.WorkStatsReport{
		Totals: model.WorkStatsTotals{},
		Actors: []model.WorkStatsActor{},
	}

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= $1 AND created_at < $2) AS day_count,
			COUNT(*) FILTER (WHERE created_at >= $3 AND created_at < $4) AS week_count,
			COUNT(*) FILTER (WHERE created_at >= $5 AND created_at < $6) AS month_count,
			COUNT(*) AS all_count
		FROM bot_moderation_actions
	`,
		bounds.DayStart,
		bounds.DayEnd,
		bounds.WeekStart,
		bounds.WeekEnd,
		bounds.MonthStart,
		bounds.MonthEnd,
	).Scan(
		&report.Totals.Day,
		&report.Totals.Week,
		&report.Totals.Month,
		&report.Totals.All,
	)
	if err != nil {
		return model.WorkStatsReport{}, fmt.Errorf("aggregate total work stats: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ma.actor_tg_id,
			ma.actor_role,
			COALESCE(NULLIF(bu.username, ''), '') AS username,
			COUNT(*) FILTER (WHERE ma.created_at >= $1 AND ma.created_at < $2) AS day_count,
			COUNT(*) FILTER (WHERE ma.created_at >= $3 AND ma.created_at < $4) AS week_count,
			COUNT(*) FILTER (WHERE ma.created_at >= $5 AND ma.created_at < $6) AS month_count,
			COUNT(*) AS all_count
		FROM bot_moderation_actions ma
		LEFT JOIN bot_users bu ON bu.tg_id = ma.actor_tg_id
		GROUP BY ma.actor_tg_id, ma.actor_role, bu.username
		ORDER BY all_count DESC, ma.actor_tg_id ASC, ma.actor_role ASC
	`,
		bounds.DayStart,
		bounds.DayEnd,
		bounds.WeekStart,
		bounds.WeekEnd,
		bounds.MonthStart,
		bounds.MonthEnd,
	)
	if err != nil {
		return model.WorkStatsReport{}, fmt.Errorf("aggregate work stats by actor: %w", err)
	}
	defer rows.Close()

	actors := make([]model.WorkStatsActor, 0, 16)
	for rows.Next() {
		var actor model.WorkStatsActor
		if err := rows.Scan(
			&actor.ActorTGID,
			&actor.ActorRole,
			&actor.Username,
			&actor.Day,
			&actor.Week,
			&actor.Month,
			&actor.All,
		); err != nil {
			return model.WorkStatsReport{}, fmt.Errorf("scan work stats by actor: %w", err)
		}
		actors = append(actors, actor)
	}
	if err := rows.Err(); err != nil {
		return model.WorkStatsReport{}, fmt.Errorf("iterate work stats by actor: %w", err)
	}

	report.Actors = actors
	return report, nil
}
