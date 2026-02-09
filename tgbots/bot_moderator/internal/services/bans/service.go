package bans

import (
	"context"
)

type Repo interface {
	Ban(context.Context, int64, string, int64) error
	Unban(context.Context, int64, int64) error
	GetBanState(context.Context, int64) (bool, string, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Ban(ctx context.Context, userID int64, reason string, updatedByTGID int64) error {
	if s.repo == nil {
		return nil
	}
	return s.repo.Ban(ctx, userID, reason, updatedByTGID)
}

func (s *Service) Unban(ctx context.Context, userID int64, updatedByTGID int64) error {
	if s.repo == nil {
		return nil
	}
	return s.repo.Unban(ctx, userID, updatedByTGID)
}

func (s *Service) GetBanState(ctx context.Context, userID int64) (bool, string, error) {
	if s.repo == nil {
		return false, "", nil
	}
	return s.repo.GetBanState(ctx, userID)
}
