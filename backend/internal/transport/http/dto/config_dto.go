package dto

type ConfigResponse struct {
	Limits    ConfigLimitsResponse  `json:"limits"`
	AntiAbuse ConfigAntiAbuse       `json:"antiabuse"`
	AdsInject ConfigAdsInject       `json:"ads_inject"`
	Filters   ConfigFiltersResponse `json:"filters"`
	GoalsMode string                `json:"goals_mode"`
	Boost     ConfigBoostResponse   `json:"boost"`
	Cities    []ConfigCityResponse  `json:"cities"`
}

type ConfigAntiAbuse struct {
	LikeMaxPerSec        int     `json:"like_max_per_sec"`
	LikeMax10Sec         int     `json:"like_max_10s"`
	LikeMaxPerMin        int     `json:"like_max_min"`
	MinCardViewMS        int     `json:"min_card_view_ms"`
	RiskDecayHours       int     `json:"risk_decay_hours"`
	CooldownStepsSec     []int   `json:"cooldown_steps_sec"`
	ShadowThreshold      int     `json:"shadow_threshold"`
	ShadowRankMultiplier float64 `json:"shadow_rank_multiplier"`
	SuspectLikeThreshold int     `json:"suspect_like_threshold"`
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
