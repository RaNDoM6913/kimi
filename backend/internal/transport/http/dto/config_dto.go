package dto

type ConfigResponse struct {
	Limits    ConfigLimitsResponse  `json:"limits"`
	AdsInject ConfigAdsInject       `json:"ads_inject"`
	Filters   ConfigFiltersResponse `json:"filters"`
	GoalsMode string                `json:"goals_mode"`
	Boost     ConfigBoostResponse   `json:"boost"`
	Cities    []ConfigCityResponse  `json:"cities"`
}

type ConfigLimitsResponse struct {
	FreeLikesPerDay int              `json:"free_likes_per_day"`
	Plus            ConfigPlusLimits `json:"plus"`
}

type ConfigPlusLimits struct {
	UnlimitedUI  bool                 `json:"unlimited_ui"`
	RateLimits   ConfigPlusRateLimits `json:"rate_limits"`
	RewindPerDay int                  `json:"rewind_per_day"`
}

type ConfigPlusRateLimits struct {
	PerMinute int `json:"per_minute"`
	Per10Sec  int `json:"per_10sec"`
}

type ConfigAdsInject struct {
	Free int `json:"free"`
	Plus int `json:"plus"`
}

type ConfigFiltersResponse struct {
	Age    ConfigRange `json:"age"`
	Radius ConfigRange `json:"radius"`
}

type ConfigRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type ConfigBoostResponse struct {
	Duration string `json:"duration"`
}

type ConfigCityResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
