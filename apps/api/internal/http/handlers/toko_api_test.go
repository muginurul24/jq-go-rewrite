package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/catalog"
)

type fakeCatalogService struct {
	providerListResponse *catalog.ProviderListResponse
	gameListResponse     *catalog.GameListResponse
	gameListV2Response   *catalog.GameListV2Response
	err                  error
	lastProviderCode     string
}

func (f *fakeCatalogService) ProviderList(_ context.Context) (*catalog.ProviderListResponse, error) {
	return f.providerListResponse, f.err
}

func (f *fakeCatalogService) GameList(_ context.Context, providerCode string) (*catalog.GameListResponse, error) {
	f.lastProviderCode = providerCode
	return f.gameListResponse, f.err
}

func (f *fakeCatalogService) GameListV2(_ context.Context, providerCode string) (*catalog.GameListV2Response, error) {
	f.lastProviderCode = providerCode
	return f.gameListV2Response, f.err
}

func TestProviderListWritesSanitizedResponse(t *testing.T) {
	t.Parallel()

	handler := NewTokoAPIHandler(nil, &fakeCatalogService{
		providerListResponse: &catalog.ProviderListResponse{
			Success: true,
			Providers: []catalog.ProviderRecord{
				{Code: "PGSOFT", Name: "PG Soft", Status: 1},
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	recorder := httptest.NewRecorder()

	handler.ProviderList(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if response["success"] != true {
		t.Fatalf("expected success=true, got %#v", response["success"])
	}

	providers, ok := response["providers"].([]any)
	if !ok || len(providers) != 1 {
		t.Fatalf("expected one provider, got %#v", response["providers"])
	}
}

func TestGameListValidatesProviderCode(t *testing.T) {
	t.Parallel()

	handler := NewTokoAPIHandler(nil, &fakeCatalogService{})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/games", strings.NewReader(`{"provider_code":""}`))
	recorder := httptest.NewRecorder()

	handler.GameList(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `"provider_code":"Provider code is required."`) {
		t.Fatalf("unexpected validation response: %s", recorder.Body.String())
	}
}

func TestGameListV2SanitizesUpstreamErrors(t *testing.T) {
	t.Parallel()

	handler := NewTokoAPIHandler(nil, &fakeCatalogService{err: catalog.ErrUpstreamFailure})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/games/v2", strings.NewReader(`{"provider_code":"PGSOFT"}`))
	recorder := httptest.NewRecorder()

	handler.GameListV2(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `Failed to get localized game list from upstream platform`) {
		t.Fatalf("unexpected error response: %s", recorder.Body.String())
	}
}

func TestGameListPassesTrimmedProviderCodeToService(t *testing.T) {
	t.Parallel()

	catalogService := &fakeCatalogService{
		gameListResponse: &catalog.GameListResponse{
			Success:      true,
			ProviderCode: "PGSOFT",
			Games: []catalog.GameRecord{
				{ID: 1, GameCode: "mahjong", GameName: "Mahjong Ways", Banner: "https://cdn.test/mahjong.png", Status: 1},
			},
		},
	}
	handler := NewTokoAPIHandler(nil, catalogService)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/games", strings.NewReader(`{"provider_code":" PGSOFT "}`))
	recorder := httptest.NewRecorder()

	handler.GameList(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if catalogService.lastProviderCode != "PGSOFT" {
		t.Fatalf("expected trimmed provider code, got %q", catalogService.lastProviderCode)
	}
}

func TestProviderListSanitizesUnexpectedErrors(t *testing.T) {
	t.Parallel()

	handler := NewTokoAPIHandler(nil, &fakeCatalogService{err: errors.New("boom")})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	recorder := httptest.NewRecorder()

	handler.ProviderList(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}

	if strings.Contains(recorder.Body.String(), "boom") {
		t.Fatalf("unexpected raw error leakage: %s", recorder.Body.String())
	}
}
