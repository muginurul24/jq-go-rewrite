package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/catalog"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
)

const (
	defaultGameLogPage    = 0
	defaultGameLogPerPage = 100
)

type LegacyPlayerAPIHandler struct {
	balanceService balanceLookupService
	playerService  playerLookupService
	nexusClient    playerNexusClient
}

type playerLookupService interface {
	FindByUsername(ctx context.Context, tokoID int64, username string) (*players.Player, error)
	UsernameMapForToko(ctx context.Context, tokoID int64) (map[string]string, error)
}

type playerNexusClient interface {
	GameLaunch(ctx context.Context, userCode string, providerCode string, lang string, gameCode *string) (*nexusggrintegration.GameLaunchResponse, error)
	MoneyInfo(ctx context.Context, userCode *string, allUsers bool) (*nexusggrintegration.MoneyInfoResponse, error)
	GetGameLog(ctx context.Context, userCode string, gameType string, start string, end string, page int, perPage int) (*nexusggrintegration.GameLogResponse, error)
}

type gameLaunchRequest struct {
	Username     string  `json:"username"`
	ProviderCode string  `json:"provider_code"`
	GameCode     *string `json:"game_code"`
	Lang         *string `json:"lang"`
}

type moneyInfoRequest struct {
	Username *string `json:"username"`
	AllUsers *bool   `json:"all_users"`
}

type gameLogRequest struct {
	Username string `json:"username"`
	GameType string `json:"game_type"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Page     *int   `json:"page"`
	PerPage  *int   `json:"perPage"`
}

type gameLogRecord struct {
	Type     any `json:"type"`
	BetMoney any `json:"bet_money"`
	WinMoney any `json:"win_money"`
	TxnID    any `json:"txn_id"`
	TxnType  any `json:"txn_type"`
}

func NewLegacyPlayerAPIHandler(
	balanceService balanceLookupService,
	playerService playerLookupService,
	nexusClient playerNexusClient,
) *LegacyPlayerAPIHandler {
	return &LegacyPlayerAPIHandler{
		balanceService: balanceService,
		playerService:  playerService,
		nexusClient:    nexusClient,
	}
}

func (h *LegacyPlayerAPIHandler) GameLaunch(w http.ResponseWriter, r *http.Request) {
	var request gameLaunchRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	username := strings.ToLower(strings.TrimSpace(request.Username))
	providerCode := strings.TrimSpace(request.ProviderCode)
	lang := "en"
	if request.Lang != nil && strings.TrimSpace(*request.Lang) != "" {
		lang = strings.TrimSpace(*request.Lang)
	}

	errorsByField := map[string]string{}
	if username == "" {
		errorsByField["username"] = "Username is required."
	} else if len(username) > 50 {
		errorsByField["username"] = "Username must not exceed 50 characters."
	}
	if providerCode == "" {
		errorsByField["provider_code"] = "Provider code is required."
	} else if len(providerCode) > 50 {
		errorsByField["provider_code"] = "Provider code must not exceed 50 characters."
	}
	if request.GameCode != nil && len(strings.TrimSpace(*request.GameCode)) > 50 {
		errorsByField["game_code"] = "Game code must not exceed 50 characters."
	}
	if len(lang) > 5 {
		errorsByField["lang"] = "Lang must not exceed 5 characters."
	}

	if len(errorsByField) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors:  errorsByField,
		})
		return
	}

	player, ok := h.playerForCurrentToko(w, r, username)
	if !ok {
		return
	}

	response, err := h.nexusClient.GameLaunch(r.Context(), player.ExtUsername, providerCode, lang, request.GameCode)
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to launch game on upstream platform")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"launch_url": response.LaunchURL,
	})
}

func (h *LegacyPlayerAPIHandler) MoneyInfo(w http.ResponseWriter, r *http.Request) {
	var request moneyInfoRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	toko, ok := auth.CurrentToko(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, authErrorResponse{Message: "Forbidden"})
		return
	}

	var (
		player   *players.Player
		username string
	)
	if request.Username != nil && strings.TrimSpace(*request.Username) != "" {
		username = strings.ToLower(strings.TrimSpace(*request.Username))
		if len(username) > 50 {
			writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
				Message: "Validation failed",
				Errors: map[string]string{
					"username": "Username must not exceed 50 characters.",
				},
			})
			return
		}

		player, ok = h.playerForCurrentToko(w, r, username)
		if !ok {
			return
		}
	}

	balance, err := h.balanceService.GetOrCreateForToko(r.Context(), toko.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to load balance"})
		return
	}

	allUsers := request.AllUsers != nil && *request.AllUsers
	var userCode *string
	if player != nil {
		userCode = &player.ExtUsername
	}

	response, err := h.nexusClient.MoneyInfo(r.Context(), userCode, allUsers)
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to get balance information from upstream platform")
		return
	}

	payload := map[string]any{
		"success": true,
		"agent": map[string]any{
			"code":    toko.Name,
			"balance": balance.NexusGGR,
		},
	}

	if response.User != nil && player != nil {
		payload["user"] = map[string]any{
			"username": player.Username,
			"balance":  toIntValue(response.User["balance"]),
		}
	}

	if len(response.UserList) > 0 {
		usernameMap, err := h.playerService.UsernameMapForToko(r.Context(), toko.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to load player mapping"})
			return
		}

		payload["user_list"] = mapMoneyInfoRecords(response.UserList, usernameMap)
	}

	writeJSON(w, http.StatusOK, payload)
}

func (h *LegacyPlayerAPIHandler) GameLog(w http.ResponseWriter, r *http.Request) {
	var request gameLogRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	username := strings.ToLower(strings.TrimSpace(request.Username))
	gameType := strings.TrimSpace(request.GameType)
	start := strings.TrimSpace(request.Start)
	end := strings.TrimSpace(request.End)
	page := defaultGameLogPage
	if request.Page != nil {
		page = *request.Page
	}
	perPage := defaultGameLogPerPage
	if request.PerPage != nil {
		perPage = *request.PerPage
	}

	errorsByField := map[string]string{}
	if username == "" {
		errorsByField["username"] = "Username is required."
	} else if len(username) > 50 {
		errorsByField["username"] = "Username must not exceed 50 characters."
	}
	if gameType == "" {
		errorsByField["game_type"] = "Game type is required."
	} else if len(gameType) > 50 {
		errorsByField["game_type"] = "Game type must not exceed 50 characters."
	}
	if start == "" {
		errorsByField["start"] = "Start is required."
	}
	if end == "" {
		errorsByField["end"] = "End is required."
	}
	if page < 0 {
		errorsByField["page"] = "Page must be at least 0."
	}
	if perPage < 1 {
		errorsByField["perPage"] = "PerPage must be at least 1."
	} else if perPage > 500 {
		errorsByField["perPage"] = "PerPage must not exceed 500."
	}

	if len(errorsByField) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors:  errorsByField,
		})
		return
	}

	player, ok := h.playerForCurrentToko(w, r, username)
	if !ok {
		return
	}

	response, err := h.nexusClient.GetGameLog(r.Context(), player.ExtUsername, gameType, start, end, page, perPage)
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to get game logs from upstream platform")
		return
	}

	logs := make([]gameLogRecord, 0, len(response.Slot))
	for _, record := range response.Slot {
		logs = append(logs, gameLogRecord{
			Type:     record["type"],
			BetMoney: record["bet_money"],
			WinMoney: record["win_money"],
			TxnID:    record["txn_id"],
			TxnType:  record["txn_type"],
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"total_count": coalesce(response.TotalCount, 0),
		"page":        coalesce(response.Page, 0),
		"perPage":     coalesce(response.PerPage, 0),
		"logs":        logs,
	})
}

func (h *LegacyPlayerAPIHandler) playerForCurrentToko(w http.ResponseWriter, r *http.Request, username string) (*players.Player, bool) {
	toko, ok := auth.CurrentToko(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, authErrorResponse{Message: "Forbidden"})
		return nil, false
	}

	player, err := h.playerService.FindByUsername(r.Context(), toko.ID, username)
	if err != nil {
		if errors.Is(err, players.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"success": false,
				"message": "Player not found",
			})
			return nil, false
		}

		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to load player"})
		return nil, false
	}

	return player, true
}

func writePlayerAPIUpstreamError(w http.ResponseWriter, err error, fallbackMessage string) {
	if errors.Is(err, catalog.ErrUpstreamFailure) {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: fallbackMessage})
		return
	}

	writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: fallbackMessage})
}

func mapMoneyInfoRecords(records []map[string]any, usernameMap map[string]string) []map[string]any {
	mapped := make([]map[string]any, 0, len(records))
	for _, record := range records {
		externalUsername, ok := record["user_code"].(string)
		if !ok {
			continue
		}

		username, exists := usernameMap[externalUsername]
		if !exists {
			continue
		}

		mapped = append(mapped, map[string]any{
			"username": username,
			"balance":  record["balance"],
		})
	}

	return mapped
}

func toIntValue(value any) any {
	switch typedValue := value.(type) {
	case int:
		return typedValue
	case int64:
		return int(typedValue)
	case int32:
		return int(typedValue)
	case float32:
		return int(typedValue)
	case float64:
		return int(typedValue)
	case string:
		trimmed := strings.TrimSpace(typedValue)
		if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return int(parsed)
		}
		if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return int(parsed)
		}
		return typedValue
	default:
		return value
	}
}

func coalesce(value any, fallback any) any {
	if value == nil {
		return fallback
	}

	return value
}
