package dto

type ProfileResponse struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	Age         int    `json:"age"`
	City        string `json:"city"`
	Zodiac      string `json:"zodiac,omitempty"`
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	Age         int    `json:"age"`
}

type ProfileCoreRequest struct {
	Birthdate  string   `json:"birthdate"`
	Gender     string   `json:"gender"`
	LookingFor string   `json:"looking_for"`
	Occupation string   `json:"occupation"`
	Education  string   `json:"education"`
	HeightCM   int      `json:"height_cm"`
	EyeColor   string   `json:"eye_color"`
	Languages  []string `json:"languages"`
	Goals      []string `json:"goals"`
}

type ProfileCoreResponse struct {
	ProfileCompleted bool `json:"profile_completed"`
}

type CandidateProfileResponse struct {
	UserID      int64                   `json:"user_id"`
	DisplayName string                  `json:"display_name"`
	Age         int                     `json:"age"`
	Zodiac      string                  `json:"zodiac,omitempty"`
	CityID      string                  `json:"city_id"`
	City        string                  `json:"city"`
	DistanceKM  *float64                `json:"distance_km,omitempty"`
	Bio         *string                 `json:"bio"`
	Occupation  string                  `json:"occupation"`
	Education   string                  `json:"education"`
	HeightCM    int                     `json:"height_cm"`
	EyeColor    string                  `json:"eye_color"`
	Languages   []string                `json:"languages"`
	Goals       []string                `json:"goals"`
	IsTravel    bool                    `json:"is_travel"`
	TravelCity  *string                 `json:"travel_city"`
	Badges      CandidateBadgesResponse `json:"badges"`
}

type CandidateBadgesResponse struct {
	IsPlus bool `json:"is_plus"`
}
