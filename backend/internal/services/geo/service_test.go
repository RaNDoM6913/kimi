package geo

import (
	"testing"

	"github.com/ivankudzin/tgapp/backend/internal/config"
)

func TestResolveNearestCity(t *testing.T) {
	svc := NewService(config.Default().Remote.Cities, nil)

	tests := []struct {
		name   string
		lat    float64
		lon    float64
		cityID string
	}{
		{name: "minsk", lat: 53.90, lon: 27.56, cityID: "minsk"},
		{name: "brest", lat: 52.10, lon: 23.75, cityID: "brest"},
		{name: "vitebsk", lat: 55.20, lon: 30.21, cityID: "vitebsk"},
		{name: "gomel", lat: 52.44, lon: 30.99, cityID: "gomel"},
		{name: "grodno", lat: 53.67, lon: 23.81, cityID: "grodno"},
		{name: "mogilev", lat: 53.90, lon: 30.34, cityID: "mogilev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			city, err := svc.ResolveNearestCity(tt.lat, tt.lon)
			if err != nil {
				t.Fatalf("resolve nearest city: %v", err)
			}
			if city.ID != tt.cityID {
				t.Fatalf("unexpected city id: got %s want %s", city.ID, tt.cityID)
			}
		})
	}
}
