DROP INDEX IF EXISTS idx_user_safety_stats_updated_at;

DROP TABLE IF EXISTS user_safety_stats;

DROP INDEX IF EXISTS idx_reports_target_created;

ALTER TABLE reports
DROP COLUMN IF EXISTS reporter_role;

ALTER TABLE reports
DROP COLUMN IF EXISTS reporter_trust_score;
