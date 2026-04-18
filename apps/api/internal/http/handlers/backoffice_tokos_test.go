package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/tokos"
)

type fakeBackofficeTokoService struct {
	listResult   *tokos.AdminListResult
	detailResult *tokos.AdminTokoRecord
	createResult *tokos.AdminTokoRecord
	updateResult *tokos.AdminTokoRecord
	regenResult  *tokos.AdminTokoRecord
	err          error
}

func (f *fakeBackofficeTokoService) ListForBackoffice(_ context.Context, _ auth.PublicUser, _ tokos.AdminListInput) (*tokos.AdminListResult, error) {
	return f.listResult, f.err
}

func (f *fakeBackofficeTokoService) FindDetailForBackoffice(_ context.Context, _ auth.PublicUser, _ int64) (*tokos.AdminTokoRecord, error) {
	return f.detailResult, f.err
}

func (f *fakeBackofficeTokoService) CreateForBackoffice(_ context.Context, _ auth.PublicUser, _ tokos.CreateInput) (*tokos.AdminTokoRecord, error) {
	return f.createResult, f.err
}

func (f *fakeBackofficeTokoService) UpdateForBackoffice(_ context.Context, _ auth.PublicUser, _ int64, _ tokos.UpdateInput) (*tokos.AdminTokoRecord, error) {
	return f.updateResult, f.err
}

func (f *fakeBackofficeTokoService) RegenerateTokenForBackoffice(_ context.Context, _ auth.PublicUser, _ int64) (*tokos.AdminTokoRecord, error) {
	return f.regenResult, f.err
}

func TestBackofficeTokosListMasksToken(t *testing.T) {
	t.Parallel()

	token := "123|abcdefghijklmnopqrstuvwxyz"
	handler := NewBackofficeTokosHandler(&fakeBackofficeTokoService{
		listResult: &tokos.AdminListResult{
			Data: []tokos.AdminTokoRecord{
				{
					ID:            7,
					UserID:        10,
					OwnerUsername: "owner-a",
					OwnerName:     "Owner A",
					Name:          "Toko Alpha",
					Token:         &token,
					IsActive:      true,
					Pending:       125000,
					Settle:        95000,
					Nexusggr:      50000,
					CreatedAt:     time.Unix(1_700_000_000, 0),
					UpdatedAt:     time.Unix(1_700_000_100, 0),
				},
			},
			Page:       1,
			PerPage:    25,
			Total:      1,
			TotalPages: 1,
			Summary: tokos.AdminTokoSummary{
				TotalTokos:    1,
				ActiveTokos:   1,
				TotalPending:  125000,
				TotalSettle:   95000,
				TotalNexusggr: 50000,
			},
			Owners: []tokos.AdminOwnerOption{
				{ID: 10, Username: "owner-a", Name: "Owner A"},
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/tokos", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 1, Role: "dev",
	}))
	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data := payload["data"].([]any)
	first := data[0].(map[string]any)
	if _, exists := first["token"]; exists {
		t.Fatalf("expected list response to omit full token, got %#v", first["token"])
	}
	if first["tokenPreview"] == nil {
		t.Fatalf("expected token preview to be present")
	}

	filters := payload["filters"].(map[string]any)
	owners := filters["owners"].([]any)
	if len(owners) != 1 {
		t.Fatalf("expected one owner option, got %#v", filters["owners"])
	}

	firstOwner := owners[0].(map[string]any)
	if firstOwner["username"] != "owner-a" || firstOwner["name"] != "Owner A" {
		t.Fatalf("expected frontend-compatible owner filter, got %#v", firstOwner)
	}
}

func TestBackofficeTokosDetailReturnsFullToken(t *testing.T) {
	t.Parallel()

	token := "55|plain-token-value"
	handler := NewBackofficeTokosHandler(&fakeBackofficeTokoService{
		detailResult: &tokos.AdminTokoRecord{
			ID:            55,
			UserID:        10,
			OwnerUsername: "owner-a",
			OwnerName:     "Owner A",
			Name:          "Toko Alpha",
			Token:         &token,
			IsActive:      true,
			CreatedAt:     time.Unix(1_700_000_000, 0),
			UpdatedAt:     time.Unix(1_700_000_100, 0),
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/tokos/55", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 1, Role: "dev",
	}))
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(request.Context()).URLParams.Add("tokoID", "55")
	recorder := httptest.NewRecorder()

	handler.Detail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), token) {
		t.Fatalf("expected full token in detail response: %s", recorder.Body.String())
	}
}

func TestBackofficeTokosCreateValidatesGlobalOwner(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeTokosHandler(&fakeBackofficeTokoService{})

	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/tokos", strings.NewReader(`{"name":"Toko Alpha","isActive":true}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 1, Role: "dev",
	}))
	recorder := httptest.NewRecorder()

	handler.Create(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `"userId":"Owner is required."`) {
		t.Fatalf("unexpected validation response: %s", recorder.Body.String())
	}
}

func TestBackofficeTokosRegenerateMapsNotFound(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeTokosHandler(&fakeBackofficeTokoService{
		err: tokos.ErrNotFound,
	})

	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/tokos/7/regenerate-token", strings.NewReader(`{}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 10, Role: "admin",
	}))
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(request.Context()).URLParams.Add("tokoID", "7")
	recorder := httptest.NewRecorder()

	handler.RegenerateToken(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestBackofficeTokosCreateMapsDuplicateCallbackTo422(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeTokosHandler(&fakeBackofficeTokoService{
		err: tokos.ErrDuplicateCallbackURL,
	})

	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/tokos", strings.NewReader(`{"userId":10,"name":"Toko Alpha","callbackUrl":"store.test/callback","isActive":true}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 1, Role: "dev",
	}))
	recorder := httptest.NewRecorder()

	handler.Create(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), "Callback URL sudah dipakai toko lain") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
