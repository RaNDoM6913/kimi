package adminhttp

import (
	"context"
	"time"

	"bot_moderator/internal/domain/enums"
	"bot_moderator/internal/domain/model"
	"bot_moderator/internal/repo/postgres"
)

type AccessUsersRepo struct {
	client *Client
	db     *postgres.BotUsersRepo
	dual   bool
}

func NewAccessUsersRepo(client *Client, db *postgres.BotUsersRepo, dual bool) *AccessUsersRepo {
	return &AccessUsersRepo{
		client: client,
		db:     db,
		dual:   dual,
	}
}

func (r *AccessUsersRepo) Upsert(ctx context.Context, user model.BotUser) error {
	err := r.client.DoJSON(ctx, "POST", "/admin/bot/users/upsert", user, nil)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.Upsert(ctx, user)
	}
	return err
}

func (r *AccessUsersRepo) ListRecent(ctx context.Context, limit int) ([]model.BotUser, error) {
	response := struct {
		Items []model.BotUser `json:"items"`
	}{}
	err := r.client.DoJSON(ctx, "GET", "/admin/bot/users/recent?limit="+intToString(limit), nil, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.ListRecent(ctx, limit)
	}
	if err != nil {
		return nil, err
	}
	return response.Items, nil
}

func (r *AccessUsersRepo) GetByTGID(ctx context.Context, tgID int64) (model.BotUser, error) {
	response := model.BotUser{}
	err := r.client.DoJSON(ctx, "GET", "/admin/bot/users/"+int64ToString(tgID), nil, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.GetByTGID(ctx, tgID)
	}
	if err != nil {
		return model.BotUser{}, err
	}
	return response, nil
}

type AccessRolesRepo struct {
	client *Client
	db     *postgres.BotRolesRepo
	dual   bool
}

func NewAccessRolesRepo(client *Client, db *postgres.BotRolesRepo, dual bool) *AccessRolesRepo {
	return &AccessRolesRepo{
		client: client,
		db:     db,
		dual:   dual,
	}
}

func (r *AccessRolesRepo) GetActiveRole(ctx context.Context, tgID int64) (enums.Role, error) {
	response := struct {
		Role string `json:"role"`
	}{}
	err := r.client.DoJSON(ctx, "GET", "/admin/bot/roles/"+int64ToString(tgID)+"/active", nil, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.GetActiveRole(ctx, tgID)
	}
	if err != nil {
		return enums.RoleNone, err
	}

	switch response.Role {
	case string(enums.RoleOwner):
		return enums.RoleOwner, nil
	case string(enums.RoleAdmin):
		return enums.RoleAdmin, nil
	case string(enums.RoleModerator):
		return enums.RoleModerator, nil
	default:
		return enums.RoleNone, nil
	}
}

func (r *AccessRolesRepo) ListActive(ctx context.Context) ([]model.BotRoleAssignment, error) {
	response := struct {
		Items []model.BotRoleAssignment `json:"items"`
	}{}
	err := r.client.DoJSON(ctx, "GET", "/admin/bot/roles/active", nil, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.ListActive(ctx)
	}
	if err != nil {
		return nil, err
	}
	return response.Items, nil
}

func (r *AccessRolesRepo) GrantRole(ctx context.Context, tgID int64, role enums.Role, grantedBy int64, grantedAt time.Time) error {
	request := map[string]interface{}{
		"tg_id":      tgID,
		"role":       role,
		"granted_by": grantedBy,
		"granted_at": grantedAt,
	}
	err := r.client.DoJSON(ctx, "POST", "/admin/bot/roles/grant", request, nil)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.GrantRole(ctx, tgID, role, grantedBy, grantedAt)
	}
	return err
}

func (r *AccessRolesRepo) RevokeRole(ctx context.Context, tgID int64, revokedAt time.Time) (bool, error) {
	request := map[string]interface{}{
		"tg_id":      tgID,
		"revoked_at": revokedAt,
	}
	response := struct {
		Revoked bool `json:"revoked"`
	}{}
	err := r.client.DoJSON(ctx, "POST", "/admin/bot/roles/revoke", request, &response)
	if shouldFallback(r.dual, err) && r.db != nil {
		return r.db.RevokeRole(ctx, tgID, revokedAt)
	}
	if err != nil {
		return false, err
	}
	return response.Revoked, nil
}
