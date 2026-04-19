package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/notifications"
)

type backofficeNotificationService interface {
	ListForBackoffice(ctx context.Context, actor auth.PublicUser, input notifications.ListInput) (*notifications.ListResult, error)
	MarkRead(ctx context.Context, actor auth.PublicUser, notificationID string) error
	MarkAllRead(ctx context.Context, actor auth.PublicUser) (int64, error)
}

type BackofficeNotificationsHandler struct {
	service backofficeNotificationService
}

func NewBackofficeNotificationsHandler(service backofficeNotificationService) *BackofficeNotificationsHandler {
	return &BackofficeNotificationsHandler{service: service}
}

func (h *BackofficeNotificationsHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	page, err := parsePositiveInt(r.URL.Query().Get("page"), 1)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid page"})
		return
	}
	perPage, err := parsePositiveInt(r.URL.Query().Get("perPage"), 20)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid perPage"})
		return
	}

	result, err := h.service.ListForBackoffice(r.Context(), actor, notifications.ListInput{
		Scope:   strings.TrimSpace(r.URL.Query().Get("scope")),
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load notifications"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": presentNotifications(result.Data),
		"meta": map[string]any{
			"page":       result.Page,
			"perPage":    result.PerPage,
			"total":      result.Total,
			"totalPages": result.TotalPages,
		},
		"summary": map[string]any{
			"total":          result.Summary.Total,
			"unread":         result.Summary.Unread,
			"unreadCritical": result.Summary.UnreadCritical,
			"unreadWarnings": result.Summary.UnreadWarnings,
			"unreadSuccess":  result.Summary.UnreadSuccess,
		},
	})
}

func (h *BackofficeNotificationsHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	notificationID := strings.TrimSpace(chi.URLParam(r, "notificationID"))
	if notificationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid notification ID"})
		return
	}

	if err := h.service.MarkRead(r.Context(), actor, notificationID); err != nil {
		status := http.StatusInternalServerError
		message := "Failed to mark notification as read"
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
			message = "Notification not found"
		}
		writeJSON(w, status, map[string]string{"message": message})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Notification marked as read"})
}

func (h *BackofficeNotificationsHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	updatedCount, err := h.service.MarkAllRead(r.Context(), actor)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to mark all notifications as read"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":      "All notifications marked as read",
		"updatedCount": updatedCount,
	})
}

func presentNotifications(records []notifications.Record) []map[string]any {
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		item := map[string]any{
			"id":        record.ID,
			"type":      record.Type,
			"title":     record.Title,
			"body":      record.Body,
			"icon":      record.Icon,
			"iconColor": record.IconColor,
			"status":    record.Status,
			"createdAt": record.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		if record.ReadAt != nil {
			item["readAt"] = record.ReadAt.UTC().Format(time.RFC3339Nano)
		}
		if record.Action != nil {
			item["action"] = map[string]any{
				"label": record.Action.Label,
				"url":   record.Action.URL,
			}
		}
		result = append(result, item)
	}
	return result
}
