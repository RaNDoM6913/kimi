package dto

type ProfileResponse struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	Age         int    `json:"age"`
	City        string `json:"city"`
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
	Zodiac     string   `json:"zodiac"`
	Languages  []string `json:"languages"`
	Goals      []string `json:"goals"`
}

type ProfileCoreResponse struct {
	ProfileCompleted bool `json:"profile_completed"`
}
