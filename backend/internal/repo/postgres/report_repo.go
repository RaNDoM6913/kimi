package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepo struct {
	pool *pgxpool.Pool
}

func NewReportRepo(pool *pgxpool.Pool) *ReportRepo {
	return &ReportRepo{pool: pool}
}

func (r *ReportRepo) Create(ctx context.Context, tx pgx.Tx, reporterUserID, targetUserID int64, reason, details string) error {
	if reporterUserID <= 0 || targetUserID <= 0 || reporterUserID == targetUserID {
		return fmt.Errorf("invalid report payload")
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("report reason is required")
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO reports (
	reporter_user_id,
	target_user_id,
	reason,
	details,
	status,
	created_at,
	updated_at
) VALUES ($1, $2, $3, $4, 'new', NOW(), NOW())
`, reporterUserID, targetUserID, strings.ToLower(strings.TrimSpace(reason)), strings.TrimSpace(details)); err != nil {
		return fmt.Errorf("create report: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO profiles (
	user_id,
	display_name,
	reports_count,
	updated_at
) VALUES ($1, '', 1, NOW())
ON CONFLICT (user_id) DO UPDATE SET
	reports_count = profiles.reports_count + 1,
	updated_at = NOW()
`, targetUserID); err != nil {
		return fmt.Errorf("increment target report counter: %w", err)
	}

	return nil
}
