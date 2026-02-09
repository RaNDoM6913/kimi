package audit

import (
	"context"
	"encoding/json"
	"time"

	"bot_moderator/internal/domain/enums"
	"bot_moderator/internal/domain/model"
)

type Repo interface {
	Save(context.Context, model.Audit) error
	ListRecent(context.Context, int) ([]model.Audit, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) LogStart(ctx context.Context, actorTGID int64, role enums.Role) error {
	if s.repo == nil {
		return nil
	}

	payload, err := json.Marshal(map[string]string{
		"role": string(role),
	})
	if err != nil {
		payload = json.RawMessage(`{}`)
	}

	entry := model.Audit{
		ActorTGID: actorTGID,
		Action:    enums.AuditActionBotStart,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
	return s.repo.Save(ctx, entry)
}

func (s *Service) LogRoleGranted(ctx context.Context, actorTGID, targetTGID int64, targetUsername string, role enums.Role) error {
	return s.logRoleChange(ctx, enums.AuditActionRoleGranted, actorTGID, targetTGID, targetUsername, role)
}

func (s *Service) LogRoleRevoked(ctx context.Context, actorTGID, targetTGID int64, targetUsername string, role enums.Role) error {
	return s.logRoleChange(ctx, enums.AuditActionRoleRevoked, actorTGID, targetTGID, targetUsername, role)
}

func (s *Service) logRoleChange(ctx context.Context, action enums.AuditAction, actorTGID, targetTGID int64, targetUsername string, role enums.Role) error {
	if s.repo == nil {
		return nil
	}

	payload, err := json.Marshal(map[string]interface{}{
		"target_tg_id":    targetTGID,
		"target_username": targetUsername,
		"role":            string(role),
		"actor_tg_id":     actorTGID,
	})
	if err != nil {
		payload = json.RawMessage(`{}`)
	}

	entry := model.Audit{
		ActorTGID: actorTGID,
		Action:    action,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
	return s.repo.Save(ctx, entry)
}

func (s *Service) LogModerationReject(ctx context.Context, actorTGID int64, targetUserID int64, moderationItemID int64, reasonCode string) error {
	if s.repo == nil {
		return nil
	}

	payload, err := json.Marshal(map[string]interface{}{
		"target_user_id":     targetUserID,
		"moderation_item_id": moderationItemID,
		"reason_code":        reasonCode,
	})
	if err != nil {
		payload = json.RawMessage(`{}`)
	}

	entry := model.Audit{
		ActorTGID: actorTGID,
		Action:    enums.AuditActionModerationReject,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
	return s.repo.Save(ctx, entry)
}

func (s *Service) LogModerationApprove(ctx context.Context, actorTGID int64, targetUserID int64, moderationItemID int64) error {
	if s.repo == nil {
		return nil
	}

	payload, err := json.Marshal(map[string]interface{}{
		"target_user_id":     targetUserID,
		"moderation_item_id": moderationItemID,
	})
	if err != nil {
		payload = json.RawMessage(`{}`)
	}

	entry := model.Audit{
		ActorTGID: actorTGID,
		Action:    enums.AuditActionModerationApprove,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
	return s.repo.Save(ctx, entry)
}

func (s *Service) LogLookup(ctx context.Context, actorTGID int64, query string, targetUserID int64) error {
	return s.logWithPayload(ctx, enums.AuditActionLookupUser, actorTGID, map[string]interface{}{
		"query":          query,
		"target_user_id": targetUserID,
	})
}

func (s *Service) LogBan(ctx context.Context, actorTGID int64, targetUserID int64, reason string) error {
	return s.logWithPayload(ctx, enums.AuditActionBanUser, actorTGID, map[string]interface{}{
		"target_user_id": targetUserID,
		"reason":         reason,
	})
}

func (s *Service) LogUnban(ctx context.Context, actorTGID int64, targetUserID int64) error {
	return s.logWithPayload(ctx, enums.AuditActionUnbanUser, actorTGID, map[string]interface{}{
		"target_user_id": targetUserID,
	})
}

func (s *Service) LogForceReview(ctx context.Context, actorTGID int64, targetUserID int64) error {
	return s.logWithPayload(ctx, enums.AuditActionForceReview, actorTGID, map[string]interface{}{
		"target_user_id": targetUserID,
	})
}

func (s *Service) LogSystemToggleRegistration(ctx context.Context, actorTGID int64, newValue bool) error {
	return s.logWithPayload(ctx, enums.AuditActionSystemToggleReg, actorTGID, map[string]interface{}{
		"new_value": newValue,
	})
}

func (s *Service) LogSystemViewUsersCount(ctx context.Context, actorTGID int64, total int64, approved int64) error {
	return s.logWithPayload(ctx, enums.AuditActionSystemViewUsers, actorTGID, map[string]interface{}{
		"total":    total,
		"approved": approved,
	})
}

func (s *Service) LogSystemViewWorkStats(ctx context.Context, actorTGID int64) error {
	return s.logWithPayload(ctx, enums.AuditActionSystemViewWork, actorTGID, map[string]interface{}{})
}

func (s *Service) ListRecent(ctx context.Context, limit int) ([]model.Audit, error) {
	if s.repo == nil {
		return []model.Audit{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListRecent(ctx, limit)
}

func (s *Service) logWithPayload(ctx context.Context, action enums.AuditAction, actorTGID int64, data map[string]interface{}) error {
	if s.repo == nil {
		return nil
	}

	payload, err := json.Marshal(data)
	if err != nil {
		payload = json.RawMessage(`{}`)
	}

	entry := model.Audit{
		ActorTGID: actorTGID,
		Action:    action,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
	return s.repo.Save(ctx, entry)
}
