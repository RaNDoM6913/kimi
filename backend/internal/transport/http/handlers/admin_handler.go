package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	userssvc "github.com/ivankudzin/tgapp/backend/internal/services/users"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type DailyMetricsReader interface {
	ListDaily(ctx context.Context, from, to time.Time) ([]pgrepo.DailyMetricRow, error)
}

type AntiAbuseDashboardReader interface {
	Summary(ctx context.Context) (redrepo.AntiAbuseSummary, error)
	Top(ctx context.Context, kind string, limit int64) ([]redrepo.OffenderItem, error)
}

type AdminHandler struct {
	users     *userssvc.Service
	telemetry *analyticsvc.Service
	metrics   DailyMetricsReader
	antiabuse AntiAbuseDashboardReader
}

func NewAdminHandler(users *userssvc.Service, telemetry *analyticsvc.Service) *AdminHandler {
	return &AdminHandler{
		users:     users,
		telemetry: telemetry,
	}
}

func (h *AdminHandler) AttachDailyMetrics(metrics DailyMetricsReader) {
	h.metrics = metrics
}

func (h *AdminHandler) AttachAntiAbuseDashboard(reader AntiAbuseDashboardReader) {
	h.antiabuse = reader
}

func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}

	httperrors.Write(w, http.StatusOK, map[string]any{
		"ok":      true,
		"user_id": identity.UserID,
		"role":    identity.Role,
	})
}

func (h *AdminHandler) UserPrivate(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.users == nil {
		writeInternal(w, "USERS_SERVICE_UNAVAILABLE", "users service is unavailable")
		return
	}

	targetUserID, ok := adminTargetUserIDFromRequest(r)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid user id")
		return
	}

	h.logViewPrivateAudit(r, identity.UserID, identity.Role, targetUserID)

	privateData, err := h.users.GetPrivate(r.Context(), targetUserID)
	if err != nil {
		switch {
		case errors.Is(err, userssvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid user id")
		case errors.Is(err, userssvc.ErrNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "NOT_FOUND",
				Message: "user not found",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to load private user data")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.AdminUserPrivateResponse{
		UserID:    privateData.UserID,
		PhoneE164: privateData.PhoneE164,
		Lat:       privateData.Lat,
		Lon:       privateData.Lon,
		LastGeoAt: privateData.LastGeoAt,
	})
}

func (h *AdminHandler) MetricsDaily(w http.ResponseWriter, r *http.Request) {
	_, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.metrics == nil {
		writeInternal(w, "METRICS_SERVICE_UNAVAILABLE", "metrics service is unavailable")
		return
	}

	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	from, ok := parseDayDate(fromRaw)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "from must be YYYY-MM-DD")
		return
	}
	to, ok := parseDayDate(toRaw)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "to must be YYYY-MM-DD")
		return
	}
	if to.Before(from) {
		writeBadRequest(w, "VALIDATION_ERROR", "to must be >= from")
		return
	}

	rows, err := h.metrics.ListDaily(r.Context(), from, to)
	if err != nil {
		writeInternal(w, "INTERNAL_ERROR", "failed to load daily metrics")
		return
	}

	items := make([]dto.AdminDailyMetricsItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.AdminDailyMetricsItem{
			DayKey:     row.DayKey.UTC().Format("2006-01-02"),
			CityID:     row.CityID,
			Gender:     row.Gender,
			LookingFor: row.LookingFor,
			Likes:      row.Likes,
			Dislikes:   row.Dislikes,
			SuperLikes: row.SuperLikes,
			Matches:    row.Matches,
			Reports:    row.Reports,
			Approved:   row.Approved,
		})
	}

	httperrors.Write(w, http.StatusOK, dto.AdminDailyMetricsResponse{
		Items: items,
	})
}

func (h *AdminHandler) AntiAbuseSummary(w http.ResponseWriter, r *http.Request) {
	_, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.antiabuse == nil {
		writeInternal(w, "ANTIABUSE_DASHBOARD_UNAVAILABLE", "antiabuse dashboard is unavailable")
		return
	}

	summary, err := h.antiabuse.Summary(r.Context())
	if err != nil {
		writeInternal(w, "INTERNAL_ERROR", "failed to load antiabuse summary")
		return
	}

	httperrors.Write(w, http.StatusOK, dto.AdminAntiAbuseSummaryResponse{
		TooFast1h:         summary.TooFast1h,
		CooldownApplied1h: summary.CooldownApplied1h,
		ShadowEnabled24h:  summary.ShadowEnabled24h,
	})
}

func (h *AdminHandler) AntiAbuseTop(w http.ResponseWriter, r *http.Request) {
	_, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.antiabuse == nil {
		writeInternal(w, "ANTIABUSE_DASHBOARD_UNAVAILABLE", "antiabuse dashboard is unavailable")
		return
	}

	kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	if kind == "" {
		kind = "user"
	}
	if kind != "user" && kind != "device" {
		writeBadRequest(w, "VALIDATION_ERROR", "kind must be user or device")
		return
	}

	limit := int64(20)
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 64)
		if err != nil || parsed <= 0 {
			writeBadRequest(w, "VALIDATION_ERROR", "limit must be a positive integer")
			return
		}
		limit = parsed
	}

	items, err := h.antiabuse.Top(r.Context(), kind, limit)
	if err != nil {
		writeInternal(w, "INTERNAL_ERROR", "failed to load antiabuse top offenders")
		return
	}

	out := make([]dto.AdminAntiAbuseTopItem, 0, len(items))
	for _, item := range items {
		out = append(out, dto.AdminAntiAbuseTopItem{
			ID:    item.ID,
			Score: item.Score,
		})
	}

	httperrors.Write(w, http.StatusOK, dto.AdminAntiAbuseTopResponse{
		Kind:  kind,
		Limit: limit,
		Items: out,
	})
}

func (h *AdminHandler) logViewPrivateAudit(r *http.Request, actorUserID int64, actorRole string, targetUserID int64) {
	if h.telemetry == nil || r == nil {
		return
	}

	props := map[string]any{
		"action":         "VIEW_PRIVATE",
		"actor_user_id":  actorUserID,
		"actor_role":     strings.TrimSpace(actorRole),
		"target_user_id": targetUserID,
	}

	actor := actorUserID
	_ = h.telemetry.IngestBatch(r.Context(), &actor, []analyticsvc.BatchEvent{
		{
			Name:  "audit_log",
			TS:    time.Now().UTC().UnixMilli(),
			Props: props,
		},
	})
}

func adminTargetUserIDFromRequest(r *http.Request) (int64, bool) {
	if r == nil {
		return 0, false
	}
	rawID := strings.TrimSpace(chi.URLParam(r, "id"))
	if rawID == "" {
		return 0, false
	}
	userID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}
	return userID, true
}

func parseDayDate(raw string) (time.Time, bool) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, false
	}
	day, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, false
	}
	return day.UTC(), true
}
