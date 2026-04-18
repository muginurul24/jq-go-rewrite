package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/app"
)

type HealthHandler struct {
	runtime *app.Runtime
}

type healthResponse struct {
	Status    string                   `json:"status"`
	Timestamp string                   `json:"timestamp"`
	Services  map[string]serviceStatus `json:"services,omitempty"`
}

type serviceStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func NewHealthHandler(runtime *app.Runtime) *HealthHandler {
	return &HealthHandler{runtime: runtime}
}

func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	services := map[string]serviceStatus{
		"postgres": {Status: "ok"},
		"redis":    {Status: "ok"},
	}

	statusCode := http.StatusOK

	if err := h.runtime.DB.Ping(ctx); err != nil {
		services["postgres"] = serviceStatus{
			Status: "error",
			Error:  err.Error(),
		}
		statusCode = http.StatusServiceUnavailable
	}

	if err := h.runtime.Redis.Ping(ctx).Err(); err != nil {
		services["redis"] = serviceStatus{
			Status: "error",
			Error:  err.Error(),
		}
		statusCode = http.StatusServiceUnavailable
	}

	status := "ok"
	if statusCode != http.StatusOK {
		status = "degraded"
	}

	writeJSON(w, statusCode, healthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Services:  services,
	})
}
