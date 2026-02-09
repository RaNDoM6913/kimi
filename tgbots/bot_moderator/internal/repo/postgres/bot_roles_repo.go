package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"bot_moderator/internal/domain/enums"
	"bot_moderator/internal/domain/model"
)

type BotRolesRepo struct {
	db *sql.DB
}

func NewBotRolesRepo(db *sql.DB) *BotRolesRepo {
	return &BotRolesRepo{db: db}
}

func (r *BotRolesRepo) GetActiveRole(ctx context.Context, tgID int64) (enums.Role, error) {
	if r.db == nil {
		return enums.RoleNone, nil
	}

	var role string
	err := r.db.QueryRowContext(ctx, `
		SELECT role
		FROM bot_roles
		WHERE tg_id = $1
		  AND revoked_at IS NULL
	`, tgID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return enums.RoleNone, nil
	}
	if err != nil {
		return enums.RoleNone, err
	}

	return normalizeRole(role), nil
}

func (r *BotRolesRepo) ListActive(ctx context.Context) ([]model.BotRoleAssignment, error) {
	if r.db == nil {
		return []model.BotRoleAssignment{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT br.tg_id,
		       br.role,
		       COALESCE(bu.username, ''),
		       COALESCE(bu.first_name, ''),
		       COALESCE(bu.last_name, '')
		FROM bot_roles br
		LEFT JOIN bot_users bu ON bu.tg_id = br.tg_id
		WHERE br.revoked_at IS NULL
		  AND br.role IN ($1, $2)
		ORDER BY br.role ASC, bu.username ASC NULLS LAST, br.tg_id ASC
	`, string(enums.RoleAdmin), string(enums.RoleModerator))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]model.BotRoleAssignment, 0, 32)
	for rows.Next() {
		var assignment model.BotRoleAssignment
		var role string
		if err := rows.Scan(
			&assignment.TgID,
			&role,
			&assignment.Username,
			&assignment.FirstName,
			&assignment.LastName,
		); err != nil {
			return nil, err
		}
		assignment.Role = normalizeRole(role)
		result = append(result, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *BotRolesRepo) GrantRole(ctx context.Context, tgID int64, role enums.Role, grantedBy int64, grantedAt time.Time) error {
	if r.db == nil {
		return nil
	}
	if grantedAt.IsZero() {
		grantedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bot_roles (tg_id, role, granted_by, granted_at, revoked_at)
		VALUES ($1, $2, $3, $4, NULL)
		ON CONFLICT (tg_id)
		DO UPDATE
		SET role = EXCLUDED.role,
		    granted_by = EXCLUDED.granted_by,
		    granted_at = EXCLUDED.granted_at,
		    revoked_at = NULL
	`, tgID, string(role), grantedBy, grantedAt)
	return err
}

func (r *BotRolesRepo) RevokeRole(ctx context.Context, tgID int64, revokedAt time.Time) (bool, error) {
	if r.db == nil {
		return false, nil
	}
	if revokedAt.IsZero() {
		revokedAt = time.Now().UTC()
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE bot_roles
		SET revoked_at = $2
		WHERE tg_id = $1
		  AND revoked_at IS NULL
	`, tgID, revokedAt)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func normalizeRole(role string) enums.Role {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case string(enums.RoleOwner):
		return enums.RoleOwner
	case string(enums.RoleAdmin):
		return enums.RoleAdmin
	case string(enums.RoleModerator):
		return enums.RoleModerator
	default:
		return enums.RoleNone
	}
}
