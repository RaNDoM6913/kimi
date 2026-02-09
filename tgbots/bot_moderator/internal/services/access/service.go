package access

import (
	"context"
	"errors"
	"time"

	"bot_moderator/internal/domain/enums"
	"bot_moderator/internal/domain/model"
)

type UsersRepo interface {
	Upsert(context.Context, model.BotUser) error
	ListRecent(context.Context, int) ([]model.BotUser, error)
	GetByTGID(context.Context, int64) (model.BotUser, error)
}

type RolesRepo interface {
	GetActiveRole(context.Context, int64) (enums.Role, error)
	ListActive(context.Context) ([]model.BotRoleAssignment, error)
	GrantRole(context.Context, int64, enums.Role, int64, time.Time) error
	RevokeRole(context.Context, int64, time.Time) (bool, error)
}

type Service struct {
	ownerTGID int64
	usersRepo UsersRepo
	rolesRepo RolesRepo
}

var ErrAccessDenied = errors.New("access denied")

func NewService(ownerTGID int64, usersRepo UsersRepo, rolesRepo RolesRepo) *Service {
	return &Service{
		ownerTGID: ownerTGID,
		usersRepo: usersRepo,
		rolesRepo: rolesRepo,
	}
}

func (s *Service) TouchUser(ctx context.Context, user model.BotUser) error {
	if s.usersRepo == nil {
		return nil
	}
	if user.LastSeenAt.IsZero() {
		user.LastSeenAt = time.Now().UTC()
	}
	return s.usersRepo.Upsert(ctx, user)
}

func (s *Service) ResolveRole(ctx context.Context, tgID int64) (enums.Role, error) {
	if s.ownerTGID != 0 && tgID == s.ownerTGID {
		return enums.RoleOwner, nil
	}

	if s.rolesRepo == nil {
		return enums.RoleNone, nil
	}

	role, err := s.rolesRepo.GetActiveRole(ctx, tgID)
	if err != nil {
		return enums.RoleNone, err
	}
	return role, nil
}

func (s *Service) CanOpenAccess(role enums.Role) bool {
	return role == enums.RoleOwner || role == enums.RoleAdmin
}

func (s *Service) CanGrantRole(actorRole, targetRole enums.Role) bool {
	switch actorRole {
	case enums.RoleOwner:
		return targetRole == enums.RoleAdmin || targetRole == enums.RoleModerator
	case enums.RoleAdmin:
		return targetRole == enums.RoleModerator
	default:
		return false
	}
}

func (s *Service) CanRevokeRole(actorRole, targetRole enums.Role) bool {
	switch actorRole {
	case enums.RoleOwner:
		return targetRole == enums.RoleAdmin || targetRole == enums.RoleModerator
	case enums.RoleAdmin:
		return targetRole == enums.RoleModerator
	default:
		return false
	}
}

func (s *Service) ListRecentUsers(ctx context.Context, limit int) ([]model.BotUser, error) {
	if s.usersRepo == nil {
		return []model.BotUser{}, nil
	}
	return s.usersRepo.ListRecent(ctx, limit)
}

func (s *Service) GetUser(ctx context.Context, tgID int64) (model.BotUser, error) {
	if s.usersRepo == nil {
		return model.BotUser{TgID: tgID}, nil
	}
	return s.usersRepo.GetByTGID(ctx, tgID)
}

func (s *Service) ListActiveAssignments(ctx context.Context) ([]model.BotRoleAssignment, error) {
	if s.rolesRepo == nil {
		return []model.BotRoleAssignment{}, nil
	}
	return s.rolesRepo.ListActive(ctx)
}

func (s *Service) GetActiveManagedRole(ctx context.Context, tgID int64) (enums.Role, error) {
	if s.rolesRepo == nil {
		return enums.RoleNone, nil
	}
	role, err := s.rolesRepo.GetActiveRole(ctx, tgID)
	if err != nil {
		return enums.RoleNone, err
	}
	if role != enums.RoleAdmin && role != enums.RoleModerator {
		return enums.RoleNone, nil
	}
	return role, nil
}

func (s *Service) GrantRole(ctx context.Context, actorTGID int64, actorRole enums.Role, targetTGID int64, targetRole enums.Role) error {
	if targetTGID == s.ownerTGID {
		return ErrAccessDenied
	}
	if !s.CanGrantRole(actorRole, targetRole) {
		return ErrAccessDenied
	}
	if s.rolesRepo == nil {
		return nil
	}
	return s.rolesRepo.GrantRole(ctx, targetTGID, targetRole, actorTGID, time.Now().UTC())
}

func (s *Service) RevokeRole(ctx context.Context, actorRole enums.Role, targetTGID int64) (enums.Role, bool, error) {
	if targetTGID == s.ownerTGID {
		return enums.RoleNone, false, ErrAccessDenied
	}
	if s.rolesRepo == nil {
		return enums.RoleNone, false, nil
	}

	targetRole, err := s.rolesRepo.GetActiveRole(ctx, targetTGID)
	if err != nil {
		return enums.RoleNone, false, err
	}
	if targetRole == enums.RoleNone {
		return enums.RoleNone, false, nil
	}
	if !s.CanRevokeRole(actorRole, targetRole) {
		return targetRole, false, ErrAccessDenied
	}

	revoked, err := s.rolesRepo.RevokeRole(ctx, targetTGID, time.Now().UTC())
	if err != nil {
		return targetRole, false, err
	}
	return targetRole, revoked, nil
}
