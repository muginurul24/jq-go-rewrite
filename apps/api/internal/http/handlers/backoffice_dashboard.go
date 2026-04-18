package handlers

import (
	"context"
	"net/http"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/dashboard"
)

type backofficeDashboardService interface {
	Overview(ctx context.Context, actor auth.PublicUser) (*dashboard.OverviewResult, error)
	OperationalPulse(ctx context.Context, actor auth.PublicUser) (*dashboard.OperationalPulseResult, error)
}

type BackofficeDashboardHandler struct {
	service backofficeDashboardService
}

func NewBackofficeDashboardHandler(service backofficeDashboardService) *BackofficeDashboardHandler {
	return &BackofficeDashboardHandler{service: service}
}

func (h *BackofficeDashboardHandler) Overview(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	result, err := h.service.Overview(r.Context(), actor)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load dashboard overview"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

func (h *BackofficeDashboardHandler) OperationalPulse(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	result, err := h.service.OperationalPulse(r.Context(), actor)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load dashboard operational pulse"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}
