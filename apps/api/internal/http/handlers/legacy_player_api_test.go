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
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/balances"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
)

type fakeBalanceLookupService struct {
	balance *balances.Balance
	err     error
}

func (f *fakeBalanceLookupService) GetOrCreateForToko(_ context.Context, _ int64) (*balances.Balance, error) {
	return f.balance, f.err
}

type fakePlayerLookupService struct {
	player       *players.Player
	usernameMap  map[string]string
	findErr      error
	usernameErr  error
	lastUsername string
}

func (f *fakePlayerLookupService) FindByUsername(_ context.Context, _ int64, username string) (*players.Player, error) {
	f.lastUsername = username
	return f.player, f.findErr
}

func (f *fakePlayerLookupService) UsernameMapForToko(_ context.Context, _ int64) (map[string]string, error) {
	return f.usernameMap, f.usernameErr
}

type fakePlayerNexusClient struct {
	gameLaunchResponse *nexusggrintegration.GameLaunchResponse
	moneyInfoResponse  *nexusggrintegration.MoneyInfoResponse
	gameLogResponse    *nexusggrintegration.GameLogResponse
	err                error
	lastUserCode       string
}

func (f *fakePlayerNexusClient) GameLaunch(_ context.Context, userCode string, _ string, _ string, _ *string) (*nexusggrintegration.GameLaunchResponse, error) {
	f.lastUserCode = userCode
	return f.gameLaunchResponse, f.err
}

func (f *fakePlayerNexusClient) MoneyInfo(_ context.Context, userCode *string, _ bool) (*nexusggrintegration.MoneyInfoResponse, error) {
	if userCode != nil {
		f.lastUserCode = *userCode
	}
	return f.moneyInfoResponse, f.err
}

func (f *fakePlayerNexusClient) GetGameLog(_ context.Context, userCode string, _ string, _ string, _ string, _ int, _ int) (*nexusggrintegration.GameLogResponse, error) {
	f.lastUserCode = userCode
	return f.gameLogResponse, f.err
}

func TestMoneyInfoMapsLocalAgentAndUserPayload(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerAPIHandler(
		&fakeBalanceLookupService{balance: &balances.Balance{NexusGGR: 500000}},
		&fakePlayerLookupService{
			player: &players.Player{Username: "shareduser", ExtUsername: "ext-a"},
		},
		&fakePlayerNexusClient{
			moneyInfoResponse: &nexusggrintegration.MoneyInfoResponse{
				Status: 1,
				User: map[string]any{
					"user_code": "ext-a",
					"balance":   json.Number("9000.87"),
				},
			},
		},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/money/info", strings.NewReader(`{"username":"SHAREDUSER"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.MoneyInfo(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	agent := response["agent"].(map[string]any)
	if agent["code"] != "Toko A" || agent["balance"] != float64(500000) {
		t.Fatalf("unexpected agent payload: %#v", agent)
	}

	user := response["user"].(map[string]any)
	if user["username"] != "shareduser" || user["balance"] != float64(9000) {
		t.Fatalf("unexpected user payload: %#v", user)
	}
}

func TestMoneyInfoMapsUserListToLocalUsernames(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerAPIHandler(
		&fakeBalanceLookupService{balance: &balances.Balance{NexusGGR: 500000}},
		&fakePlayerLookupService{
			usernameMap: map[string]string{
				"ext-a": "player-a",
			},
		},
		&fakePlayerNexusClient{
			moneyInfoResponse: &nexusggrintegration.MoneyInfoResponse{
				Status: 1,
				UserList: []map[string]any{
					{"user_code": "ext-a", "balance": 12000},
					{"user_code": "ext-missing", "balance": 99999},
				},
			},
		},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/money/info", strings.NewReader(`{"all_users":true}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.MoneyInfo(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	userList := response["user_list"].([]any)
	if len(userList) != 1 {
		t.Fatalf("expected one mapped user, got %#v", response["user_list"])
	}
}

func TestGameLaunchReturnsNotFoundWhenPlayerMissing(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerAPIHandler(
		nil,
		&fakePlayerLookupService{findErr: players.ErrNotFound},
		&fakePlayerNexusClient{},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/game/launch", strings.NewReader(`{"username":"missing","provider_code":"PGSOFT"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.GameLaunch(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `Player not found`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestGameLogSanitizesRecords(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerAPIHandler(
		nil,
		&fakePlayerLookupService{
			player: &players.Player{Username: "shareduser", ExtUsername: "ext-a"},
		},
		&fakePlayerNexusClient{
			gameLogResponse: &nexusggrintegration.GameLogResponse{
				Status:     1,
				TotalCount: 1,
				Page:       0,
				PerPage:    100,
				Slot: []map[string]any{
					{
						"type":      "slot",
						"bet_money": 1000,
						"win_money": 1500,
						"txn_id":    "TXN-1",
						"txn_type":  "credit",
						"secret":    "hidden",
					},
				},
			},
		},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/game/log", strings.NewReader(`{"username":"shareduser","game_type":"slot","start":"2026-04-04 00:00:00","end":"2026-04-04 23:59:59"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.GameLog(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if strings.Contains(recorder.Body.String(), "secret") {
		t.Fatalf("unexpected raw secret leakage: %s", recorder.Body.String())
	}
}

func TestGameLaunchSanitizesUpstreamErrors(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerAPIHandler(
		nil,
		&fakePlayerLookupService{
			player: &players.Player{Username: "shareduser", ExtUsername: "ext-a"},
		},
		&fakePlayerNexusClient{err: errors.New("boom")},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/game/launch", strings.NewReader(`{"username":"shareduser","provider_code":"PGSOFT"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.GameLaunch(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}

	if strings.Contains(recorder.Body.String(), "boom") {
		t.Fatalf("unexpected raw error leakage: %s", recorder.Body.String())
	}
}
