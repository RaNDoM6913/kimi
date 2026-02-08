package profiles

import (
	"context"
	"errors"
	"testing"
	"time"
)

type saveCall struct {
	birthdate        time.Time
	gender           string
	lookingFor       string
	occupation       string
	education        string
	heightCM         int
	eyeColor         string
	zodiac           string
	languages        []string
	goals            []string
	profileCompleted bool
}

type fakeStore struct {
	lastCall *saveCall
}

func (f *fakeStore) SaveCore(
	_ context.Context,
	_ int64,
	birthdate time.Time,
	gender string,
	lookingFor string,
	occupation string,
	education string,
	heightCM int,
	eyeColor string,
	zodiac string,
	languages []string,
	goals []string,
	profileCompleted bool,
) error {
	f.lastCall = &saveCall{
		birthdate:        birthdate,
		gender:           gender,
		lookingFor:       lookingFor,
		occupation:       occupation,
		education:        education,
		heightCM:         heightCM,
		eyeColor:         eyeColor,
		zodiac:           zodiac,
		languages:        languages,
		goals:            goals,
		profileCompleted: profileCompleted,
	}
	return nil
}

func TestUpdateCoreAgeRejected(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	svc.now = func() time.Time {
		return time.Date(2026, time.February, 8, 12, 0, 0, 0, time.UTC)
	}

	_, err := svc.UpdateCore(context.Background(), 1, CoreInput{
		Birthdate:  time.Date(2010, time.February, 9, 0, 0, 0, 0, time.UTC),
		Gender:     "male",
		LookingFor: "female",
		Occupation: "engineer",
		Education:  "higher",
		HeightCM:   180,
		EyeColor:   "brown",
		Zodiac:     "aries",
		Languages:  []string{"ru"},
		Goals:      []string{"relationship"},
	})
	if !errors.Is(err, ErrAgeRejected) {
		t.Fatalf("expected ErrAgeRejected, got %v", err)
	}
}

func TestUpdateCoreValidationErrorForLists(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	svc.now = func() time.Time {
		return time.Date(2026, time.February, 8, 12, 0, 0, 0, time.UTC)
	}

	_, err := svc.UpdateCore(context.Background(), 1, CoreInput{
		Birthdate:  time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC),
		Gender:     "male",
		LookingFor: "female",
		Occupation: "engineer",
		Education:  "higher",
		HeightCM:   180,
		EyeColor:   "brown",
		Zodiac:     "aries",
		Languages:  []string{"xx"},
		Goals:      []string{"relationship"},
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestUpdateCoreMarksProfileCompleted(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	svc.now = func() time.Time {
		return time.Date(2026, time.February, 8, 12, 0, 0, 0, time.UTC)
	}

	completed, err := svc.UpdateCore(context.Background(), 42, CoreInput{
		Birthdate:  time.Date(1995, time.March, 12, 0, 0, 0, 0, time.UTC),
		Gender:     "male",
		LookingFor: "female",
		Occupation: "engineer",
		Education:  "higher",
		HeightCM:   180,
		EyeColor:   "brown",
		Zodiac:     "aries",
		Languages:  []string{"ru", "en", "ru"},
		Goals:      []string{"relationship", "chat"},
	})
	if err != nil {
		t.Fatalf("update core: %v", err)
	}
	if !completed {
		t.Fatalf("expected completed=true")
	}
	if store.lastCall == nil {
		t.Fatalf("expected save call")
	}
	if !store.lastCall.profileCompleted {
		t.Fatalf("expected profileCompleted=true in store")
	}
	if len(store.lastCall.languages) != 2 {
		t.Fatalf("expected deduplicated languages length=2, got %d", len(store.lastCall.languages))
	}
}
