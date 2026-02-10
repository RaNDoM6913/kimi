package cleanup

import (
	"context"
	"testing"
	"time"
)

func TestRunClearsExactGeoOlderThanRetention(t *testing.T) {
	now := time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)

	latOld := 53.9
	lonOld := 27.56
	latFresh := 52.1
	lonFresh := 23.73

	cleaner := &fakeGeoCleaner{
		profiles: []geoProfile{
			{
				LastGeoAt: ptrTime(now.Add(-49 * time.Hour)),
				LastLat:   &latOld,
				LastLon:   &lonOld,
			},
			{
				LastGeoAt: ptrTime(now.Add(-47 * time.Hour)),
				LastLat:   &latFresh,
				LastLon:   &lonFresh,
			},
		},
	}

	job := New()
	job.now = func() time.Time { return now }
	job.AttachExactGeoCleanup(cleaner, 48*time.Hour)

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("run cleanup job: %v", err)
	}

	if cleaner.profiles[0].LastLat != nil || cleaner.profiles[0].LastLon != nil {
		t.Fatalf("expected old exact coordinates to be cleared")
	}
	if cleaner.profiles[1].LastLat == nil || cleaner.profiles[1].LastLon == nil {
		t.Fatalf("expected fresh exact coordinates to remain")
	}
}

type geoProfile struct {
	LastGeoAt *time.Time
	LastLat   *float64
	LastLon   *float64
}

type fakeGeoCleaner struct {
	profiles []geoProfile
}

func (f *fakeGeoCleaner) ClearExactGeoOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	var affected int64
	for i := range f.profiles {
		profile := &f.profiles[i]
		if profile.LastGeoAt == nil {
			continue
		}
		if profile.LastGeoAt.Before(cutoff) {
			if profile.LastLat != nil || profile.LastLon != nil {
				profile.LastLat = nil
				profile.LastLon = nil
				affected++
			}
		}
	}
	return affected, nil
}

func ptrTime(v time.Time) *time.Time {
	value := v.UTC()
	return &value
}
