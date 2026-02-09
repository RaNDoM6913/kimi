package system

import (
	"context"
	"testing"
)

type fakeRepo struct {
	registrationEnabled bool
	totalUsers          int64
	approvedUsers       int64
}

func (r *fakeRepo) GetRegistrationEnabled(_ context.Context) (bool, error) {
	return r.registrationEnabled, nil
}

func (r *fakeRepo) ToggleRegistration(_ context.Context, _ int64) (bool, error) {
	r.registrationEnabled = !r.registrationEnabled
	return r.registrationEnabled, nil
}

func (r *fakeRepo) GetUsersCount(_ context.Context) (int64, int64, error) {
	return r.totalUsers, r.approvedUsers, nil
}

func TestSystemServiceAppFlagsAndUsersCount(t *testing.T) {
	repo := &fakeRepo{
		registrationEnabled: true,
		totalUsers:          42,
		approvedUsers:       25,
	}
	svc := NewService(repo)

	enabled, err := svc.GetRegistrationEnabled(context.Background())
	if err != nil {
		t.Fatalf("get registration flag: %v", err)
	}
	if !enabled {
		t.Fatalf("expected registration enabled=true, got false")
	}

	newValue, err := svc.ToggleRegistration(context.Background(), 999001)
	if err != nil {
		t.Fatalf("toggle registration: %v", err)
	}
	if newValue {
		t.Fatalf("expected toggled registration=false, got true")
	}

	count, err := svc.GetUsersCount(context.Background())
	if err != nil {
		t.Fatalf("get users count: %v", err)
	}
	if count.Total != 42 || count.Approved != 25 {
		t.Fatalf("unexpected users count: %+v", count)
	}
}
