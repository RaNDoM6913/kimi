package model

import (
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/domain/enums"
)

type Profile struct {
	UserID              int64                  `json:"user_id"`
	DisplayName         string                 `json:"display_name"`
	Bio                 string                 `json:"bio"`
	Birthdate           *time.Time             `json:"birthdate"`
	Age                 int                    `json:"age"`
	Gender              string                 `json:"gender"`
	LookingFor          string                 `json:"looking_for"`
	Occupation          string                 `json:"occupation"`
	Education           string                 `json:"education"`
	HeightCM            int                    `json:"height_cm"`
	EyeColor            string                 `json:"eye_color"`
	Zodiac              string                 `json:"zodiac"`
	Languages           []string               `json:"languages"`
	Goals               []string               `json:"goals"`
	ProfileCompleted    bool                   `json:"profile_completed"`
	CityID              string                 `json:"city_id"`
	City                string                 `json:"city"`
	LastGeoAt           *time.Time             `json:"last_geo_at"`
	LastLat             float64                `json:"last_lat"`
	LastLon             float64                `json:"last_lon"`
	GeoMandatoryDone    bool                   `json:"geo_mandatory_done"`
	PhotosCount         int                    `json:"photos_count"`
	ReportsCount        int                    `json:"reports_count"`
	HasCircle           bool                   `json:"has_circle"`
	ModerationStatus    enums.ModerationStatus `json:"moderation_status"`
	ModerationETABucket string                 `json:"moderation_eta_bucket"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}
