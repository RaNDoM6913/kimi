package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"bot_moderator/internal/domain/enums"
	"bot_moderator/internal/domain/model"
	"bot_moderator/internal/services/access"
	"bot_moderator/internal/ui"
)

type stubUsersRepo struct {
	called int
	last   model.BotUser
	err    error
}

func (s *stubUsersRepo) Upsert(_ context.Context, user model.BotUser) error {
	s.called++
	s.last = user
	return s.err
}

func (s *stubUsersRepo) ListRecent(_ context.Context, _ int) ([]model.BotUser, error) {
	return []model.BotUser{}, nil
}

func (s *stubUsersRepo) GetByTGID(_ context.Context, tgID int64) (model.BotUser, error) {
	return model.BotUser{TgID: tgID}, nil
}

type stubRolesRepo struct {
	role enums.Role
	err  error
}

func (s *stubRolesRepo) GetActiveRole(_ context.Context, _ int64) (enums.Role, error) {
	return s.role, s.err
}

func (s *stubRolesRepo) ListActive(_ context.Context) ([]model.BotRoleAssignment, error) {
	return []model.BotRoleAssignment{}, nil
}

func (s *stubRolesRepo) GrantRole(_ context.Context, _ int64, _ enums.Role, _ int64, _ time.Time) error {
	return nil
}

func (s *stubRolesRepo) RevokeRole(_ context.Context, _ int64, _ time.Time) (bool, error) {
	return true, nil
}

func TestAccessResolveRole(t *testing.T) {
	svc := access.NewService(999, nil, &stubRolesRepo{role: enums.RoleModerator})

	ownerRole, err := svc.ResolveRole(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ownerRole != enums.RoleOwner {
		t.Fatalf("expected owner role, got %s", ownerRole)
	}

	moderatorRole, err := svc.ResolveRole(context.Background(), 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if moderatorRole != enums.RoleModerator {
		t.Fatalf("expected moderator role from bot_roles, got %s", moderatorRole)
	}
}

func TestAccessResolveRoleRepoError(t *testing.T) {
	svc := access.NewService(0, nil, &stubRolesRepo{err: errors.New("db error")})

	_, err := svc.ResolveRole(context.Background(), 1000)
	if err == nil {
		t.Fatal("expected resolve role error")
	}
}

func TestAccessTouchUser(t *testing.T) {
	usersRepo := &stubUsersRepo{}
	svc := access.NewService(0, usersRepo, nil)

	err := svc.TouchUser(context.Background(), model.BotUser{TgID: 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usersRepo.called != 1 {
		t.Fatalf("expected one upsert call, got %d", usersRepo.called)
	}
	if usersRepo.last.LastSeenAt.IsZero() {
		t.Fatal("expected last_seen_at to be set")
	}
}

func TestStartRenderingForAllRoles(t *testing.T) {
	testCases := []struct {
		name        string
		role        enums.Role
		expectText  string
		mustHave    []string
		mustNotHave []string
		expectEmpty bool
	}{
		{
			name:        "none",
			role:        enums.RoleNone,
			expectText:  "У вас нет доступа к этому боту",
			expectEmpty: true,
		},
		{
			name:        "moderator",
			role:        enums.RoleModerator,
			mustHave:    []string{"Dating App", "Приступить к модерации"},
			mustNotHave: []string{"Access", "Find user", "System", "Work Stats"},
		},
		{
			name:        "admin",
			role:        enums.RoleAdmin,
			mustHave:    []string{"Dating App", "Приступить к модерации", "Access", "Find user"},
			mustNotHave: []string{"System", "Work Stats"},
		},
		{
			name:     "owner",
			role:     enums.RoleOwner,
			mustHave: []string{"Dating App", "Приступить к модерации", "Access", "Find user", "History", "System", "Work Stats"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			text, menu := ui.RenderStart(tc.role)
			if text == "" {
				t.Fatalf("role %s: empty text", tc.role)
			}
			if tc.expectText != "" && text != tc.expectText {
				t.Fatalf("role %s: unexpected text %q", tc.role, text)
			}

			if tc.expectEmpty {
				if len(menu) != 0 {
					t.Fatalf("role %s: expected empty menu", tc.role)
				}
				return
			}

			if len(menu) == 0 {
				t.Fatalf("role %s: empty menu", tc.role)
			}

			for _, expected := range tc.mustHave {
				if !menuContains(menu, expected) {
					t.Fatalf("role %s: missing menu item %q", tc.role, expected)
				}
			}

			for _, denied := range tc.mustNotHave {
				if menuContains(menu, denied) {
					t.Fatalf("role %s: unexpected menu item %q", tc.role, denied)
				}
			}
		})
	}
}

func menuContains(menu [][]string, expected string) bool {
	for _, row := range menu {
		for _, item := range row {
			if item == expected {
				return true
			}
		}
	}
	return false
}
