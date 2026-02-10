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

func (r *ReportRepo) Create(
	ctx context.Context,
	tx pgx.Tx,
	reporterUserID, targetUserID int64,
	reason, details string,
	reporterTrustScore int,
	reporterRole string,
) error {
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
	reporter_trust_score,
	reporter_role,
	status,
	created_at,
	updated_at
) VALUES ($1, $2, $3, $4, $5, $6, 'new', NOW(), NOW())
`, reporterUserID, targetUserID, strings.ToLower(strings.TrimSpace(reason)), strings.TrimSpace(details), reporterTrustScore, normalizeReporterRole(reporterRole)); err != nil {
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

	if _, err := tx.Exec(ctx, `
INSERT INTO user_safety_stats (
	user_id,
	reports_24h,
	reports_7d,
	updated_at
) VALUES (
	$1,
	(SELECT COUNT(*)::INT FROM reports WHERE target_user_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'),
	(SELECT COUNT(*)::INT FROM reports WHERE target_user_id = $1 AND created_at >= NOW() - INTERVAL '7 days'),
	NOW()
)
ON CONFLICT (user_id) DO UPDATE SET
	reports_24h = EXCLUDED.reports_24h,
	reports_7d = EXCLUDED.reports_7d,
	updated_at = NOW()
`, targetUserID); err != nil {
		return fmt.Errorf("upsert user safety stats: %w", err)
	}

	return nil
}

func normalizeReporterRole(role string) string {
	value := strings.ToLower(strings.TrimSpace(role))
	if value == "" {
		return "user"
	}
	return value
}
