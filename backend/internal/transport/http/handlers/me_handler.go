package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/config"
	antiabusesvc "github.com/ivankudzin/tgapp/backend/internal/services/antiabuse"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type MeHandler struct {
	remote    config.RemoteConfig
	antiabuse *antiabusesvc.Service
	now       func() time.Time
}

func NewMeHandler(remote config.RemoteConfig, antiabuse *antiabusesvc.Service) *MeHandler {
	return &MeHandler{
		remote:    remote,
		antiabuse: antiabuse,
		now:       time.Now,
	}
}

func (h *MeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}

	now := h.now().UTC()
	loc := time.UTC
	if tz := h.remote.MeDefaults.Timezone; tz != "" {
		if loaded, err := time.LoadLocation(tz); err == nil {
			loc = loaded
		}
	}

	plusUntil := (*time.Time)(nil)
	if h.remote.MeDefaults.IsPlus && h.remote.MeDefaults.PlusDuration > 0 {
		v := now.Add(h.remote.MeDefaults.PlusDuration)
		plusUntil = &v
	}

	incognitoUntil := (*time.Time)(nil)
	if h.remote.MeDefaults.Entitlements.IncognitoDuration > 0 {
		v := now.Add(h.remote.MeDefaults.Entitlements.IncognitoDuration)
		incognitoUntil = &v
	}

	likesLeft := h.remote.Limits.FreeLikesPerDay
	if h.remote.MeDefaults.IsPlus && h.remote.Limits.PlusUnlimitedUI {
		likesLeft = -1
	}

	resetAt := nextLocalMidnight(now, loc)
	tooFast := (*int64)(nil)
	if h.remote.MeDefaults.TooFastRetryAfterSec > 0 {
		v := h.remote.MeDefaults.TooFastRetryAfterSec
		tooFast = &v
	}

	username := fmt.Sprintf("%s%d", h.remote.MeDefaults.UsernamePrefix, identity.UserID)
	if h.remote.MeDefaults.UsernamePrefix == "" {
		username = fmt.Sprintf("u%d", identity.UserID)
	}

	antiAbuseState := dto.MeAntiAbuseState{
		RiskScore:     0,
		CooldownUntil: nil,
		ShadowEnabled: nil,
	}
	if h.antiabuse != nil {
		state, err := h.antiabuse.ApplyDecay(r.Context(), identity.UserID, now)
		if err == nil {
			antiAbuseState.RiskScore = float64(state.RiskScore)
			antiAbuseState.CooldownUntil = state.CooldownUntil
			if state.Exists {
				v := state.ShadowEnabled
				antiAbuseState.ShadowEnabled = &v
			}
		}
	}

	httperrors.Write(w, http.StatusOK, dto.MeResponse{
		User: dto.MeUserPublicResponse{
			ID:        identity.UserID,
			TGID:      identity.UserID,
			Username:  username,
			Role:      identity.Role,
			Zodiac:    nil,
			IsPlus:    h.remote.MeDefaults.IsPlus,
			PlusUntil: plusUntil,
			CityID:    h.remote.MeDefaults.CityID,
		},
		ModerationStatus: h.remote.MeDefaults.ModerationStatus,
		Entitlements: dto.MeEntitlementsResponse{
			SuperLikeCredits:      h.remote.MeDefaults.Entitlements.SuperLikeCredits,
			BoostCredits:          h.remote.MeDefaults.Entitlements.BoostCredits,
			RevealCredits:         h.remote.MeDefaults.Entitlements.RevealCredits,
			MessageWoMatchCredits: h.remote.MeDefaults.Entitlements.MessageWoMatchCredits,
			IncognitoUntil:        incognitoUntil,
		},
		Quota: dto.MeQuotaSnapshotResponse{
			LikesLeft:         likesLeft,
			ResetAt:           resetAt,
			TooFastRetryAfter: tooFast,
		},
		AntiAbuseState: antiAbuseState,
	})
}

func nextLocalMidnight(now time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	next := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, loc)
	return next.UTC()
}
