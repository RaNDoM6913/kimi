package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type SystemRepo struct {
	db *sql.DB
}

func NewSystemRepo(db *sql.DB) *SystemRepo {
	return &SystemRepo{db: db}
}

func (r *SystemRepo) GetRegistrationEnabled(ctx context.Context) (bool, error) {
	if r.db == nil {
		return true, nil
	}

	if err := r.ensureRegistrationFlag(ctx); err != nil {
		return true, err
	}

	var enabled bool
	err := r.db.QueryRowContext(ctx, `
		SELECT value_bool
		FROM app_flags
		WHERE key = 'registration_enabled'
		LIMIT 1
	`).Scan(&enabled)
	if err != nil {
		return true, fmt.Errorf("get registration flag: %w", err)
	}
	return enabled, nil
}

func (r *SystemRepo) ToggleRegistration(ctx context.Context, updatedByTGID int64) (bool, error) {
	if r.db == nil {
		return true, nil
	}
	if updatedByTGID == 0 {
		return false, fmt.Errorf("invalid updated_by_tg_id")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin toggle registration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO app_flags (key, value_bool, updated_at, updated_by_tg_id)
		VALUES ('registration_enabled', TRUE, NOW(), NULL)
		ON CONFLICT (key) DO NOTHING
	`)
	if err != nil {
		return false, fmt.Errorf("ensure registration flag in toggle: %w", err)
	}

	var newValue bool
	err = tx.QueryRowContext(ctx, `
		UPDATE app_flags
		SET value_bool = NOT value_bool,
		    updated_at = NOW(),
		    updated_by_tg_id = $1
		WHERE key = 'registration_enabled'
		RETURNING value_bool
	`, updatedByTGID).Scan(&newValue)
	if err != nil {
		return false, fmt.Errorf("toggle registration flag: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit toggle registration transaction: %w", err)
	}
	return newValue, nil
}

func (r *SystemRepo) GetUsersCount(ctx context.Context) (int64, int64, error) {
	if r.db == nil {
		return 0, 0, nil
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return 0, 0, fmt.Errorf("count users total: %w", err)
	}

	var approved int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles WHERE approved = TRUE`).Scan(&approved); err != nil {
		return 0, 0, fmt.Errorf("count users approved: %w", err)
	}

	return total, approved, nil
}

func (r *SystemRepo) ensureRegistrationFlag(ctx context.Context) error {
	if r.db == nil {
		return nil
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_flags (key, value_bool, updated_at, updated_by_tg_id)
		VALUES ('registration_enabled', TRUE, NOW(), NULL)
		ON CONFLICT (key) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("ensure registration flag: %w", err)
	}

	var exists bool
	err = r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM app_flags WHERE key = 'registration_enabled')
	`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check registration flag existence: %w", err)
	}
	if !exists {
		return errors.New("registration flag is not available")
	}
	return nil
}
