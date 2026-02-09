package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/config"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type ConfigHandler struct {
	remote config.RemoteConfig
}

func NewConfigHandler(remote config.RemoteConfig) *ConfigHandler {
	return &ConfigHandler{remote: remote}
}

func (h *ConfigHandler) Handle(w http.ResponseWriter, _ *http.Request) {
	cities := make([]dto.ConfigCityResponse, 0, len(h.remote.Cities))
	for _, city := range h.remote.Cities {
		cities = append(cities, dto.ConfigCityResponse{ID: city.ID, Name: city.Name})
	}

	httperrors.Write(w, http.StatusOK, dto.ConfigResponse{
		Limits: dto.ConfigLimitsResponse{
			FreeLikesPerDay: h.remote.Limits.FreeLikesPerDay,
			Plus: dto.ConfigPlusLimits{
				UnlimitedUI: h.remote.Limits.PlusUnlimitedUI,
				RateLimits: dto.ConfigPlusRateLimits{
					PerMinute: h.remote.Limits.PlusRatePerMinute,
					Per10Sec:  h.remote.Limits.PlusRatePer10Seconds,
				},
				RewindPerDay: h.remote.Limits.PlusRewindsPerDay,
			},
		},
		AntiAbuse: dto.ConfigAntiAbuse{
			LikeMaxPerSec:        h.remote.AntiAbuse.LikeMaxPerSec,
			LikeMax10Sec:         h.remote.AntiAbuse.LikeMax10Sec,
			LikeMaxPerMin:        h.remote.AntiAbuse.LikeMaxPerMin,
			MinCardViewMS:        h.remote.AntiAbuse.MinCardViewMS,
			RiskDecayHours:       h.remote.AntiAbuse.RiskDecayHours,
			CooldownStepsSec:     h.remote.AntiAbuse.CooldownStepsSec,
			ShadowThreshold:      h.remote.AntiAbuse.ShadowThreshold,
			ShadowRankMultiplier: h.remote.AntiAbuse.ShadowRankMultiplier,
			SuspectLikeThreshold: h.remote.AntiAbuse.SuspectLikeThreshold,
		},
		AdsInject: dto.ConfigAdsInject{
			Free: h.remote.AdsInject.FreeEvery,
			Plus: h.remote.AdsInject.PlusEvery,
		},
		Filters: dto.ConfigFiltersResponse{
			Age: dto.ConfigRange{
				Min: h.remote.Filters.AgeMin,
				Max: h.remote.Filters.AgeMax,
			},
			Radius: dto.ConfigRange{
				Min: h.remote.Filters.RadiusDefaultKM,
				Max: h.remote.Filters.RadiusMaxKM,
			},
		},
		GoalsMode: h.remote.GoalsMode,
		Boost: dto.ConfigBoostResponse{
			Duration: formatDuration(h.remote.Boost.Duration),
		},
		Cities: cities,
	})
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return d.String()
}
