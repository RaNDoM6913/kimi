package rules

import (
	"testing"
	"time"
)

func TestDayKeyUsesTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Minsk")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	utc := time.Date(2026, 2, 8, 21, 30, 0, 0, time.UTC)
	got := DayKey(utc, loc)
	want := "2026-02-09"
	if got != want {
		t.Fatalf("unexpected day key: got %s want %s", got, want)
	}
}

func TestDayKeyDefaultsToUTC(t *testing.T) {
	utc := time.Date(2026, 2, 8, 23, 59, 59, 0, time.UTC)
	got := DayKey(utc, nil)
	want := "2026-02-08"
	if got != want {
		t.Fatalf("unexpected day key: got %s want %s", got, want)
	}
}

func TestNextResetAtUsesTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Minsk")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	now := time.Date(2026, 2, 8, 21, 30, 0, 0, time.UTC) // 00:30 local, Feb 9
	got := NextResetAt(now, loc)
	want := time.Date(2026, 2, 9, 21, 0, 0, 0, time.UTC) // midnight local Feb 10
	if !got.Equal(want) {
		t.Fatalf("unexpected reset_at: got %s want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}
