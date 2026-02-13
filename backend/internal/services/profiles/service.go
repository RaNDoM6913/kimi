package profiles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/domain/rules"
)

var (
	ErrValidation  = errors.New("validation error")
	ErrAgeRejected = errors.New("age rejected")
)

type ProfileStore interface {
	SaveCore(
		ctx context.Context,
		userID int64,
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
	) error
}

type Service struct {
	store ProfileStore
	now   func() time.Time
}

type CoreInput struct {
	Birthdate  time.Time
	Gender     string
	LookingFor string
	Occupation string
	Education  string
	HeightCM   int
	EyeColor   string
	Zodiac     string
	Languages  []string
	Goals      []string
}

func NewService(store ProfileStore) *Service {
	return &Service{
		store: store,
		now:   time.Now,
	}
}

func (s *Service) UpdateCore(ctx context.Context, userID int64, in CoreInput) (bool, error) {
	if userID <= 0 {
		return false, fmt.Errorf("invalid user id: %w", ErrValidation)
	}
	if s.store == nil {
		return false, fmt.Errorf("profile store is nil")
	}

	normalized, err := normalizeAndValidateInput(s.now(), in)
	if err != nil {
		return false, err
	}

	profileCompleted := isProfileCompleted(normalized)
	if err := s.store.SaveCore(
		ctx,
		userID,
		normalized.Birthdate,
		normalized.Gender,
		normalized.LookingFor,
		normalized.Occupation,
		normalized.Education,
		normalized.HeightCM,
		normalized.EyeColor,
		normalized.Zodiac,
		normalized.Languages,
		normalized.Goals,
		profileCompleted,
	); err != nil {
		return false, fmt.Errorf("save profile core: %w", err)
	}

	return profileCompleted, nil
}

func normalizeAndValidateInput(now time.Time, in CoreInput) (CoreInput, error) {
	if in.Birthdate.IsZero() {
		return CoreInput{}, fmt.Errorf("birthdate is required: %w", ErrValidation)
	}

	age := ageYears(in.Birthdate, now)
	if age < 18 {
		return CoreInput{}, ErrAgeRejected
	}

	out := CoreInput{
		Birthdate:  in.Birthdate,
		Gender:     strings.ToLower(strings.TrimSpace(in.Gender)),
		LookingFor: strings.ToLower(strings.TrimSpace(in.LookingFor)),
		Occupation: strings.TrimSpace(in.Occupation),
		Education:  strings.TrimSpace(in.Education),
		HeightCM:   in.HeightCM,
		EyeColor:   strings.ToLower(strings.TrimSpace(in.EyeColor)),
		Zodiac:     rules.ZodiacFromBirthdate(in.Birthdate),
	}

	if out.Gender == "" || out.LookingFor == "" || out.Occupation == "" || out.Education == "" || out.EyeColor == "" || out.Zodiac == "" {
		return CoreInput{}, fmt.Errorf("required fields are missing: %w", ErrValidation)
	}
	if out.HeightCM < 100 || out.HeightCM > 250 {
		return CoreInput{}, fmt.Errorf("invalid height_cm: %w", ErrValidation)
	}

	languages, err := normalizeList(in.Languages, allowedLanguages)
	if err != nil {
		return CoreInput{}, err
	}
	goals, err := normalizeList(in.Goals, allowedGoals)
	if err != nil {
		return CoreInput{}, err
	}
	out.Languages = languages
	out.Goals = goals

	return out, nil
}

func normalizeList(values []string, allowed map[string]struct{}) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("list is required: %w", ErrValidation)
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			return nil, fmt.Errorf("empty list item: %w", ErrValidation)
		}
		if _, ok := allowed[normalized]; !ok {
			return nil, fmt.Errorf("value %q is not allowed: %w", normalized, ErrValidation)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("list is required: %w", ErrValidation)
	}
	return result, nil
}

func isProfileCompleted(in CoreInput) bool {
	return !in.Birthdate.IsZero() &&
		in.Gender != "" &&
		in.LookingFor != "" &&
		in.Occupation != "" &&
		in.Education != "" &&
		in.HeightCM >= 100 &&
		in.EyeColor != "" &&
		in.Zodiac != "" &&
		len(in.Languages) > 0 &&
		len(in.Goals) > 0
}

func ageYears(birthdate time.Time, now time.Time) int {
	b := birthdate.UTC()
	n := now.UTC()

	years := n.Year() - b.Year()
	if n.Month() < b.Month() || (n.Month() == b.Month() && n.Day() < b.Day()) {
		years--
	}

	return years
}

var allowedLanguages = map[string]struct{}{
	"ru": {},
	"be": {},
	"en": {},
	"pl": {},
}

var allowedGoals = map[string]struct{}{
	"relationship": {},
	"friendship":   {},
	"chat":         {},
	"networking":   {},
}
