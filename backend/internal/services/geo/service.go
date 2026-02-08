package geo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/config"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNoCities   = errors.New("no cities configured")
)

type ProfileLocationSaver interface {
	SaveLocation(ctx context.Context, userID int64, cityID string, lat, lon float64, at time.Time) error
}

type City struct {
	ID   string
	Name string
	Lat  float64
	Lon  float64
}

type Service struct {
	cities []City
	saver  ProfileLocationSaver
	now    func() time.Time
}

func NewService(cities []config.CityConfig, saver ProfileLocationSaver) *Service {
	mapped := make([]City, 0, len(cities))
	for _, city := range cities {
		if strings.TrimSpace(city.ID) == "" || strings.TrimSpace(city.Name) == "" {
			continue
		}
		mapped = append(mapped, City{ID: city.ID, Name: city.Name, Lat: city.Lat, Lon: city.Lon})
	}

	return &Service{
		cities: mapped,
		saver:  saver,
		now:    time.Now,
	}
}

func (s *Service) UpdateProfileLocation(ctx context.Context, userID int64, lat, lon float64) (City, error) {
	if userID <= 0 {
		return City{}, fmt.Errorf("invalid user id: %w", ErrValidation)
	}
	if err := validateCoordinates(lat, lon); err != nil {
		return City{}, err
	}

	city, err := s.ResolveNearestCity(lat, lon)
	if err != nil {
		return City{}, err
	}

	if s.saver != nil {
		if err := s.saver.SaveLocation(ctx, userID, city.ID, lat, lon, s.now()); err != nil {
			return City{}, err
		}
	}

	return city, nil
}

func (s *Service) ResolveNearestCity(lat, lon float64) (City, error) {
	if err := validateCoordinates(lat, lon); err != nil {
		return City{}, err
	}
	if len(s.cities) == 0 {
		return City{}, ErrNoCities
	}

	nearest := s.cities[0]
	bestDistance := haversineKM(lat, lon, nearest.Lat, nearest.Lon)
	for _, city := range s.cities[1:] {
		distance := haversineKM(lat, lon, city.Lat, city.Lon)
		if distance < bestDistance {
			bestDistance = distance
			nearest = city
		}
	}

	return nearest, nil
}

func validateCoordinates(lat, lon float64) error {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return fmt.Errorf("invalid coordinates: %w", ErrValidation)
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return fmt.Errorf("coordinates out of range: %w", ErrValidation)
	}
	return nil
}

func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKM = 6371.0

	toRad := func(v float64) float64 { return v * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKM * c
}
