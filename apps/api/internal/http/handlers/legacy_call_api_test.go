package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
)

type fakeCallPlayerLookupService struct {
	player      *players.Player
	usernameMap map[string]string
	findErr     error
	usernameErr error
}

func (f *fakeCallPlayerLookupService) FindByUsername(_ context.Context, _ int64, _ string) (*players.Player, error) {
	return f.player, f.findErr
}

func (f *fakeCallPlayerLookupService) UsernameMapForToko(_ context.Context, _ int64) (map[string]string, error) {
	return f.usernameMap, f.usernameErr
}

type fakeCallNexusClient struct {
	callPlayersResponse     *nexusggrintegration.CallPlayersResponse
	callListResponse        *nexusggrintegration.CallListResponse
	callApplyResponse       *nexusggrintegration.CallApplyResponse
	callHistoryResponse     *nexusggrintegration.CallHistoryResponse
	callCancelResponse      *nexusggrintegration.CallCancelResponse
	controlRtpResponse      *nexusggrintegration.ControlRtpResponse
	controlUsersRtpResponse *nexusggrintegration.ControlRtpResponse
	err                     error
	lastUserCode            string
	lastUserCodes           []string
}

func (f *fakeCallNexusClient) CallPlayers(_ context.Context) (*nexusggrintegration.CallPlayersResponse, error) {
	return f.callPlayersResponse, f.err
}

func (f *fakeCallNexusClient) CallList(_ context.Context, _ string, _ string) (*nexusggrintegration.CallListResponse, error) {
	return f.callListResponse, f.err
}

func (f *fakeCallNexusClient) CallApply(_ context.Context, _ string, _ string, userCode string, _ int, _ int) (*nexusggrintegration.CallApplyResponse, error) {
	f.lastUserCode = userCode
	return f.callApplyResponse, f.err
}

func (f *fakeCallNexusClient) CallHistory(_ context.Context, _ int, _ int) (*nexusggrintegration.CallHistoryResponse, error) {
	return f.callHistoryResponse, f.err
}

func (f *fakeCallNexusClient) CallCancel(_ context.Context, _ int) (*nexusggrintegration.CallCancelResponse, error) {
	return f.callCancelResponse, f.err
}

func (f *fakeCallNexusClient) ControlRtp(_ context.Context, _ string, userCode string, _ float64) (*nexusggrintegration.ControlRtpResponse, error) {
	f.lastUserCode = userCode
	return f.controlRtpResponse, f.err
}

func (f *fakeCallNexusClient) ControlUsersRtp(_ context.Context, userCodes []string, _ float64) (*nexusggrintegration.ControlRtpResponse, error) {
	f.lastUserCodes = userCodes
	return f.controlUsersRtpResponse, f.err
}

func TestCallPlayersFiltersToCurrentTokoUsers(t *testing.T) {
	t.Parallel()

	handler := NewLegacyCallAPIHandler(
		&fakeCallPlayerLookupService{
			usernameMap: map[string]string{
				"ext-a": "player-a",
			},
		},
		&fakeCallNexusClient{
			callPlayersResponse: &nexusggrintegration.CallPlayersResponse{
				Status: 1,
				Data: []map[string]any{
					{"user_code": "ext-a", "provider_code": "PG", "game_code": "G1", "bet": 100},
					{"user_code": "ext-b", "provider_code": "PG", "game_code": "G2", "bet": 200},
				},
			},
		},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/call/players", nil)
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.CallPlayers(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "user_code") {
		t.Fatalf("unexpected raw user_code leakage: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "player-a") {
		t.Fatalf("expected mapped username in response: %s", recorder.Body.String())
	}
}

func TestCallApplyReturnsNotFoundWhenPlayerMissing(t *testing.T) {
	t.Parallel()

	handler := NewLegacyCallAPIHandler(
		&fakeCallPlayerLookupService{findErr: players.ErrNotFound},
		&fakeCallNexusClient{},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/call/apply", strings.NewReader(`{"provider_code":"PG","game_code":"G1","username":"missing","call_rtp":97,"call_type":1}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.CallApply(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestCallHistoryMapsAndSanitizesRecords(t *testing.T) {
	t.Parallel()

	handler := NewLegacyCallAPIHandler(
		&fakeCallPlayerLookupService{
			usernameMap: map[string]string{
				"ext-a": "player-a",
			},
		},
		&fakeCallNexusClient{
			callHistoryResponse: &nexusggrintegration.CallHistoryResponse{
				Status: 1,
				Data: []map[string]any{
					{"id": 7, "user_code": "ext-a", "status": "pending", "secret": "hidden"},
					{"id": 8, "user_code": "ext-b", "status": "success"},
				},
			},
		},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/call/history", strings.NewReader(`{"offset":0,"limit":50}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.CallHistory(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "secret") {
		t.Fatalf("unexpected raw secret leakage: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "player-a") {
		t.Fatalf("expected mapped username in response: %s", recorder.Body.String())
	}
}

func TestControlUsersRtpReturnsNotFoundWhenAnyPlayerMissing(t *testing.T) {
	t.Parallel()

	handler := NewLegacyCallAPIHandler(
		&fakeCallPlayerLookupService{
			usernameMap: map[string]string{
				"ext-a": "player-a",
			},
		},
		&fakeCallNexusClient{},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/control/users-rtp", strings.NewReader(`{"user_codes":["player-a","player-b"],"rtp":97.5}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.ControlUsersRtp(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestControlRtpUsesExternalUsername(t *testing.T) {
	t.Parallel()

	client := &fakeCallNexusClient{
		controlRtpResponse: &nexusggrintegration.ControlRtpResponse{
			Status:     1,
			ChangedRTP: 98.2,
		},
	}

	handler := NewLegacyCallAPIHandler(
		&fakeCallPlayerLookupService{
			player: &players.Player{Username: "player-a", ExtUsername: "ext-a"},
		},
		client,
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/control/rtp", strings.NewReader(`{"provider_code":"PG","username":"PLAYER-A","rtp":98.2}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.ControlRtp(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if client.lastUserCode != "ext-a" {
		t.Fatalf("expected ext username ext-a, got %q", client.lastUserCode)
	}
}

func TestCallListSanitizesFields(t *testing.T) {
	t.Parallel()

	handler := NewLegacyCallAPIHandler(
		&fakeCallPlayerLookupService{},
		&fakeCallNexusClient{
			callListResponse: &nexusggrintegration.CallListResponse{
				Status: 1,
				Calls: []map[string]any{
					{"rtp": 95, "call_type": "Common Free", "secret": "hidden"},
				},
			},
		},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/call/list", strings.NewReader(`{"provider_code":"PG","game_code":"G1"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.CallList(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	callList := response["calls"].([]any)
	call := callList[0].(map[string]any)
	if _, exists := call["secret"]; exists {
		t.Fatalf("unexpected raw field leakage: %#v", call)
	}
}

func TestCallPlayersSanitizesUpstreamErrors(t *testing.T) {
	t.Parallel()

	handler := NewLegacyCallAPIHandler(
		&fakeCallPlayerLookupService{},
		&fakeCallNexusClient{err: errors.New("boom")},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/call/players", nil)
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.CallPlayers(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "boom") {
		t.Fatalf("unexpected raw error leakage: %s", recorder.Body.String())
	}
}
