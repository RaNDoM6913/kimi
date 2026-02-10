package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	userssvc "github.com/ivankudzin/tgapp/backend/internal/services/users"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
)

func TestAdminUserPrivateWritesAuditLog(t *testing.T) {
	store := &auditStoreStub{}
	telemetry := analyticsvc.NewService(store, analyticsvc.Config{MaxBatchSize: 100})
	handler := NewAdminHandler(userssvc.NewService(nil, nil, nil), telemetry)

	req := httptest.NewRequest(http.MethodGet, "/admin/users/42/private", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 100,
		SID:    "sid-100",
		Role:   "OWNER",
	}))
	req = req.WithContext(withURLParam(req.Context(), "id", "42"))

	rr := httptest.NewRecorder()
	handler.UserPrivate(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusNotFound)
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.events))
	}
	if store.events[0].Name != "audit_log" {
		t.Fatalf("unexpected audit event name: %q", store.events[0].Name)
	}

	action, _ := store.events[0].Props["action"].(string)
	if action != "VIEW_PRIVATE" {
		t.Fatalf("unexpected action: %q", action)
	}
	target, _ := store.events[0].Props["target_user_id"].(int64)
	if target != 42 {
		t.Fatalf("unexpected target_user_id: %d", target)
	}
}

func TestAdminHealthReturnsIdentity(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/health", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 7,
		SID:    "sid-7",
		Role:   "SUPPORT",
	}))
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestAdminMetricsDailyReturnsRows(t *testing.T) {
	handler := NewAdminHandler(nil, nil)
	handler.AttachDailyMetrics(dailyMetricsReaderStub{
		rows: []pgrepo.DailyMetricRow{
			{
				DayKey:     time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
				CityID:     "minsk",
				Gender:     "male",
				LookingFor: "female",
				Likes:      12,
				Dislikes:   7,
				SuperLikes: 2,
				Matches:    1,
				Reports:    0,
				Approved:   3,
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/metrics/daily?from=2026-02-10&to=2026-02-10", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 7,
		SID:    "sid-7",
		Role:   "SUPPORT",
	}))
	rr := httptest.NewRecorder()

	handler.MetricsDaily(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	var response dto.AdminDailyMetricsResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected one metrics row, got %d", len(response.Items))
	}
	if response.Items[0].DayKey != "2026-02-10" || response.Items[0].Likes != 12 {
		t.Fatalf("unexpected metrics payload: %+v", response.Items[0])
	}
}

func TestAdminAntiAbuseSummaryReturnsCounters(t *testing.T) {
	handler := NewAdminHandler(nil, nil)
	handler.AttachAntiAbuseDashboard(antiAbuseDashboardStub{
		summary: redrepo.AntiAbuseSummary{
			TooFast1h:         11,
			CooldownApplied1h: 9,
			ShadowEnabled24h:  3,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/antiabuse/summary", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 1,
		SID:    "sid-1",
		Role:   "OWNER",
	}))
	rr := httptest.NewRecorder()

	handler.AntiAbuseSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}
	var response dto.AdminAntiAbuseSummaryResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.TooFast1h != 11 || response.CooldownApplied1h != 9 || response.ShadowEnabled24h != 3 {
		t.Fatalf("unexpected summary payload: %+v", response)
	}
}

func TestAdminAntiAbuseTopReturnsItems(t *testing.T) {
	handler := NewAdminHandler(nil, nil)
	handler.AttachAntiAbuseDashboard(antiAbuseDashboardStub{
		top: []redrepo.OffenderItem{
			{ID: "42", Score: 7},
			{ID: "77", Score: 4},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/antiabuse/top?kind=user&limit=2", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: 1,
		SID:    "sid-1",
		Role:   "SUPPORT",
	}))
	rr := httptest.NewRecorder()

	handler.AntiAbuseTop(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}
	var response dto.AdminAntiAbuseTopResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Kind != "user" || response.Limit != 2 || len(response.Items) != 2 {
		t.Fatalf("unexpected top payload: %+v", response)
	}
}

type auditStoreStub struct {
	userIDs []*int64
	events  []pgrepo.EventWriteRecord
}

type dailyMetricsReaderStub struct {
	rows []pgrepo.DailyMetricRow
	err  error
}

func (s dailyMetricsReaderStub) ListDaily(_ context.Context, _, _ time.Time) ([]pgrepo.DailyMetricRow, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rows, nil
}

type antiAbuseDashboardStub struct {
	summary redrepo.AntiAbuseSummary
	top     []redrepo.OffenderItem
	err     error
}

func (s antiAbuseDashboardStub) Summary(context.Context) (redrepo.AntiAbuseSummary, error) {
	if s.err != nil {
		return redrepo.AntiAbuseSummary{}, s.err
	}
	return s.summary, nil
}

func (s antiAbuseDashboardStub) Top(context.Context, string, int64) ([]redrepo.OffenderItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.top, nil
}

func (s *auditStoreStub) InsertBatch(_ context.Context, userID *int64, events []pgrepo.EventWriteRecord) error {
	s.userIDs = append(s.userIDs, userID)
	for _, event := range events {
		cloned := pgrepo.EventWriteRecord{
			Name:       event.Name,
			OccurredAt: event.OccurredAt,
			Props:      make(map[string]any, len(event.Props)),
		}
		for key, value := range event.Props {
			cloned.Props[key] = value
		}
		s.events = append(s.events, cloned)
	}
	return nil
}

func withURLParam(ctx context.Context, key, value string) context.Context {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
}
