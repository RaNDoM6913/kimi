package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	modsvc "github.com/ivankudzin/tgapp/backend/internal/services/moderation"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
)

func TestAdminBotRejectReasonsUnauthorizedWithoutBotContext(t *testing.T) {
	handler := NewAdminBotModerationHandler(modsvc.NewService(nil, nil, nil, nil), nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/bot/mod/reject-reasons", nil)
	rr := httptest.NewRecorder()

	handler.RejectReasons(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got=%d want=%d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAdminBotRejectReasonsReturnsItems(t *testing.T) {
	handler := NewAdminBotModerationHandler(modsvc.NewService(nil, nil, nil, nil), nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/bot/mod/reject-reasons", nil)
	ctx := req.Context()
	ctx = authsvc.WithActorIsBot(ctx, true)
	ctx = authsvc.WithActorTGID(ctx, 777)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.RejectReasons(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d", rr.Code, http.StatusOK)
	}

	var response dto.AdminBotModerationRejectReasonsResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) == 0 {
		t.Fatalf("expected non-empty reject reasons list")
	}

	var hasOther bool
	for _, item := range response.Items {
		if item.ReasonCode == "OTHER" {
			hasOther = true
			if item.ReasonText == "" || item.RequiredFixStep == "" {
				t.Fatalf("OTHER template is incomplete: %+v", item)
			}
			break
		}
	}
	if !hasOther {
		t.Fatalf("expected OTHER reason code in response")
	}
}
