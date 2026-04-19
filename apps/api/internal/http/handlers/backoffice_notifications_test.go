package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/notifications"
)

type fakeBackofficeNotificationService struct {
	listResult    *notifications.ListResult
	markAllResult int64
	err           error
}

func (f *fakeBackofficeNotificationService) ListForBackoffice(_ context.Context, _ auth.PublicUser, _ notifications.ListInput) (*notifications.ListResult, error) {
	return f.listResult, f.err
}

func (f *fakeBackofficeNotificationService) MarkRead(_ context.Context, _ auth.PublicUser, _ string) error {
	return f.err
}

func (f *fakeBackofficeNotificationService) MarkAllRead(_ context.Context, _ auth.PublicUser) (int64, error) {
	return f.markAllResult, f.err
}

func TestBackofficeNotificationsListReturnsSummary(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	handler := NewBackofficeNotificationsHandler(&fakeBackofficeNotificationService{
		listResult: &notifications.ListResult{
			Data: []notifications.Record{
				{
					ID:        "2fd5da60-e990-4c92-a259-0ab73f4a2d9a",
					Type:      notifications.TypeWithdrawalRequestedDevNotification,
					Title:     "Withdrawal Pending",
					Body:      "Username demo-owner toko Demo Toko baru saja melakukan withdraw dengan status pending.",
					Icon:      "heroicon-o-banknotes",
					IconColor: "warning",
					Status:    "warning",
					CreatedAt: now,
				},
			},
			Page:       1,
			PerPage:    20,
			Total:      1,
			TotalPages: 1,
			Summary: notifications.ListSummary{
				Total:          1,
				Unread:         1,
				UnreadCritical: 1,
				UnreadWarnings: 1,
				UnreadSuccess:  0,
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/notifications?scope=all&page=1&perPage=20", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 1, Role: "dev"}))
	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `"unreadCritical":1`) {
		t.Fatalf("expected unreadCritical in response: %s", body)
	}
	if !strings.Contains(body, `"title":"Withdrawal Pending"`) {
		t.Fatalf("expected notification title in response: %s", body)
	}
}

func TestBackofficeNotificationsMarkReadMapsNotFound(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeNotificationsHandler(&fakeBackofficeNotificationService{
		err: pgx.ErrNoRows,
	})

	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/notifications/fake/read", strings.NewReader(`{}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 1, Role: "dev"}))
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(request.Context()).URLParams.Add("notificationID", "2fd5da60-e990-4c92-a259-0ab73f4a2d9a")
	recorder := httptest.NewRecorder()

	handler.MarkRead(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestBackofficeNotificationsMarkAllReadReturnsUpdatedCount(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeNotificationsHandler(&fakeBackofficeNotificationService{
		markAllResult: 3,
	})

	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/notifications/read-all", strings.NewReader(`{}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 1, Role: "dev"}))
	recorder := httptest.NewRecorder()

	handler.MarkAllRead(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"updatedCount":3`) {
		t.Fatalf("expected updated count in response: %s", recorder.Body.String())
	}
}
