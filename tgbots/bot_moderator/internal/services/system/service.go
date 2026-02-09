package system

import "context"

type Repo interface {
	GetRegistrationEnabled(context.Context) (bool, error)
	ToggleRegistration(context.Context, int64) (bool, error)
	GetUsersCount(context.Context) (int64, int64, error)
}

type UsersCount struct {
	Total    int64
	Approved int64
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetRegistrationEnabled(ctx context.Context) (bool, error) {
	if s.repo == nil {
		return true, nil
	}
	return s.repo.GetRegistrationEnabled(ctx)
}

func (s *Service) ToggleRegistration(ctx context.Context, actorTGID int64) (bool, error) {
	if s.repo == nil {
		return true, nil
	}
	return s.repo.ToggleRegistration(ctx, actorTGID)
}

func (s *Service) GetUsersCount(ctx context.Context) (UsersCount, error) {
	if s.repo == nil {
		return UsersCount{}, nil
	}

	total, approved, err := s.repo.GetUsersCount(ctx)
	if err != nil {
		return UsersCount{}, err
	}
	return UsersCount{
		Total:    total,
		Approved: approved,
	}, nil
}
