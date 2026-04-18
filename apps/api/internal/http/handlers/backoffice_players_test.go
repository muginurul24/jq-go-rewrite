package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
)

type fakeBackofficePlayersService struct {
	listResult      *players.AdminListResult
	moneyInfoResult *players.MoneyInfoResult
	err             error
}

func (f *fakeBackofficePlayersService) ListForBackoffice(_ context.Context, _ auth.PublicUser, _ players.AdminListInput) (*players.AdminListResult, error) {
	return f.listResult, f.err
}

func (f *fakeBackofficePlayersService) MoneyInfoForBackoffice(_ context.Context, _ auth.PublicUser, _ int64, _ players.MoneyInfoClient) (*players.MoneyInfoResult, error) {
	return f.moneyInfoResult, f.err
}

type fakeMoneyInfoClient struct{}

func (f *fakeMoneyInfoClient) MoneyInfo(_ context.Context, _ *string, _ bool) (*nexusggrintegration.MoneyInfoResponse, error) {
	return &nexusggrintegration.MoneyInfoResponse{Status: 1}, nil
}

func TestBackofficePlayersListReturnsFiltersAndRows(t *testing.T) {
	t.Parallel()

	handler := NewBackofficePlayersHandler(&fakeBackofficePlayersService{
		listResult: &players.AdminListResult{
			Data: []players.AdminPlayerRecord{
				{
					ID:            9,
					Username:      "shareduser",
					ExtUsername:   "01ext",
					TokoID:        7,
					TokoName:      "Toko Alpha",
					OwnerUsername: "owner-a",
					CreatedAt:     time.Unix(1_700_000_000, 0),
					UpdatedAt:     time.Unix(1_700_000_100, 0),
				},
			},
			Page:       1,
			PerPage:    25,
			Total:      1,
			TotalPages: 1,
			Tokos: []players.AdminTokoOption{
				{ID: 7, Name: "Toko Alpha"},
			},
		},
	}, &fakeMoneyInfoClient{})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/players", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 10, Role: "admin",
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
	if len(data) != 1 {
		t.Fatalf("expected one player row, got %#v", payload["data"])
	}

	filters := payload["filters"].(map[string]any)
	if len(filters["tokos"].([]any)) != 1 {
		t.Fatalf("expected one toko option, got %#v", filters["tokos"])
	}

	firstFilter := filters["tokos"].([]any)[0].(map[string]any)
	if firstFilter["name"] != "Toko Alpha" {
		t.Fatalf("expected frontend-compatible toko filter, got %#v", firstFilter)
	}
}

func TestBackofficePlayersMoneyInfoReturnsShape(t *testing.T) {
	t.Parallel()

	handler := NewBackofficePlayersHandler(&fakeBackofficePlayersService{
		moneyInfoResult: &players.MoneyInfoResult{
			PlayerID:    9,
			Username:    "shareduser",
			ExtUsername: "01ext",
			TokoName:    "Toko Alpha",
			Balance:     125000,
			CheckedAt:   time.Unix(1_700_000_000, 0),
		},
	}, &fakeMoneyInfoClient{})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/players/9/money-info", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 10, Role: "admin",
	}))
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(request.Context()).URLParams.Add("playerID", "9")
	recorder := httptest.NewRecorder()

	handler.MoneyInfo(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `"balance":125000`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestBackofficePlayersMoneyInfoMapsUnavailableTo502(t *testing.T) {
	t.Parallel()

	handler := NewBackofficePlayersHandler(&fakeBackofficePlayersService{
		err: players.ErrMoneyInfoUnavailable,
	}, &fakeMoneyInfoClient{})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/players/9/money-info", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 10, Role: "admin",
	}))
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(request.Context()).URLParams.Add("playerID", "9")
	recorder := httptest.NewRecorder()

	handler.MoneyInfo(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", recorder.Code)
	}
}

func TestBackofficePlayersListSanitizesUnexpectedErrors(t *testing.T) {
	t.Parallel()

	handler := NewBackofficePlayersHandler(&fakeBackofficePlayersService{
		err: errors.New("boom"),
	}, &fakeMoneyInfoClient{})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/players", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 10, Role: "admin",
	}))
	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}

	if strings.Contains(recorder.Body.String(), "boom") {
		t.Fatalf("unexpected raw error leakage: %s", recorder.Body.String())
	}
}
