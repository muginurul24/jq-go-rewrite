package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
)

const (
	defaultCallHistoryOffset = 0
	defaultCallHistoryLimit  = 50
)

type LegacyCallAPIHandler struct {
	playerService callPlayerLookupService
	nexusClient   callNexusClient
}

type callPlayerLookupService interface {
	FindByUsername(ctx context.Context, tokoID int64, username string) (*players.Player, error)
	UsernameMapForToko(ctx context.Context, tokoID int64) (map[string]string, error)
}

type callNexusClient interface {
	CallPlayers(ctx context.Context) (*nexusggrintegration.CallPlayersResponse, error)
	CallList(ctx context.Context, providerCode string, gameCode string) (*nexusggrintegration.CallListResponse, error)
	CallApply(ctx context.Context, providerCode string, gameCode string, userCode string, callRTP int, callType int) (*nexusggrintegration.CallApplyResponse, error)
	CallHistory(ctx context.Context, offset int, limit int) (*nexusggrintegration.CallHistoryResponse, error)
	CallCancel(ctx context.Context, callID int) (*nexusggrintegration.CallCancelResponse, error)
	ControlRtp(ctx context.Context, providerCode string, userCode string, rtp float64) (*nexusggrintegration.ControlRtpResponse, error)
	ControlUsersRtp(ctx context.Context, userCodes []string, rtp float64) (*nexusggrintegration.ControlRtpResponse, error)
}

type callListRequest struct {
	ProviderCode string `json:"provider_code"`
	GameCode     string `json:"game_code"`
}

type callApplyRequest struct {
	ProviderCode string `json:"provider_code"`
	GameCode     string `json:"game_code"`
	Username     string `json:"username"`
	CallRTP      *int   `json:"call_rtp"`
	CallType     *int   `json:"call_type"`
}

type callHistoryRequest struct {
	Offset *int `json:"offset"`
	Limit  *int `json:"limit"`
}

type callCancelRequest struct {
	CallID *int `json:"call_id"`
}

type controlRtpRequest struct {
	ProviderCode string   `json:"provider_code"`
	Username     string   `json:"username"`
	RTP          *float64 `json:"rtp"`
}

type controlUsersRtpRequest struct {
	UserCodes []string `json:"user_codes"`
	RTP       *float64 `json:"rtp"`
}

func NewLegacyCallAPIHandler(playerService callPlayerLookupService, nexusClient callNexusClient) *LegacyCallAPIHandler {
	return &LegacyCallAPIHandler{
		playerService: playerService,
		nexusClient:   nexusClient,
	}
}

func (h *LegacyCallAPIHandler) CallPlayers(w http.ResponseWriter, r *http.Request) {
	toko, ok := auth.CurrentToko(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, authErrorResponse{Message: "Forbidden"})
		return
	}

	response, err := h.nexusClient.CallPlayers(r.Context())
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to get active players from upstream platform")
		return
	}

	usernameMap, err := h.playerService.UsernameMapForToko(r.Context(), toko.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to load player mapping"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    mapCallPlayerRecords(response.Data, usernameMap),
	})
}

func (h *LegacyCallAPIHandler) CallList(w http.ResponseWriter, r *http.Request) {
	var request callListRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	providerCode := strings.TrimSpace(request.ProviderCode)
	gameCode := strings.TrimSpace(request.GameCode)
	errorsByField := map[string]string{}
	if providerCode == "" {
		errorsByField["provider_code"] = "Provider code is required."
	} else if len(providerCode) > 50 {
		errorsByField["provider_code"] = "Provider code must not exceed 50 characters."
	}
	if gameCode == "" {
		errorsByField["game_code"] = "Game code is required."
	} else if len(gameCode) > 50 {
		errorsByField["game_code"] = "Game code must not exceed 50 characters."
	}
	if len(errorsByField) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors:  errorsByField,
		})
		return
	}

	response, err := h.nexusClient.CallList(r.Context(), providerCode, gameCode)
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to get call list from upstream platform")
		return
	}

	calls := make([]map[string]any, 0, len(response.Calls))
	for _, call := range response.Calls {
		calls = append(calls, map[string]any{
			"rtp":       call["rtp"],
			"call_type": call["call_type"],
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"calls":   calls,
	})
}

func (h *LegacyCallAPIHandler) CallApply(w http.ResponseWriter, r *http.Request) {
	var request callApplyRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	providerCode := strings.TrimSpace(request.ProviderCode)
	gameCode := strings.TrimSpace(request.GameCode)
	username := strings.ToLower(strings.TrimSpace(request.Username))
	errorsByField := map[string]string{}
	if providerCode == "" {
		errorsByField["provider_code"] = "Provider code is required."
	} else if len(providerCode) > 50 {
		errorsByField["provider_code"] = "Provider code must not exceed 50 characters."
	}
	if gameCode == "" {
		errorsByField["game_code"] = "Game code is required."
	} else if len(gameCode) > 50 {
		errorsByField["game_code"] = "Game code must not exceed 50 characters."
	}
	if username == "" {
		errorsByField["username"] = "Username is required."
	} else if len(username) > 50 {
		errorsByField["username"] = "Username must not exceed 50 characters."
	}
	if request.CallRTP == nil {
		errorsByField["call_rtp"] = "Call RTP is required."
	}
	if request.CallType == nil {
		errorsByField["call_type"] = "Call type is required."
	} else if *request.CallType != 1 && *request.CallType != 2 {
		errorsByField["call_type"] = "Call type must be one of: 1, 2."
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

	response, err := h.nexusClient.CallApply(r.Context(), providerCode, gameCode, player.ExtUsername, *request.CallRTP, *request.CallType)
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to apply call on upstream platform")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"called_money": response.CalledMoney,
	})
}

func (h *LegacyCallAPIHandler) CallHistory(w http.ResponseWriter, r *http.Request) {
	var request callHistoryRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	offset := defaultCallHistoryOffset
	if request.Offset != nil {
		offset = *request.Offset
	}
	limit := defaultCallHistoryLimit
	if request.Limit != nil {
		limit = *request.Limit
	}

	errorsByField := map[string]string{}
	if offset < 0 {
		errorsByField["offset"] = "Offset must be at least 0."
	}
	if limit < 1 {
		errorsByField["limit"] = "Limit must be at least 1."
	} else if limit > 500 {
		errorsByField["limit"] = "Limit must not exceed 500."
	}
	if len(errorsByField) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors:  errorsByField,
		})
		return
	}

	toko, ok := auth.CurrentToko(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, authErrorResponse{Message: "Forbidden"})
		return
	}

	response, err := h.nexusClient.CallHistory(r.Context(), offset, limit)
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to get call history from upstream platform")
		return
	}

	usernameMap, err := h.playerService.UsernameMapForToko(r.Context(), toko.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to load player mapping"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    mapCallHistoryRecords(response.Data, usernameMap),
	})
}

func (h *LegacyCallAPIHandler) CallCancel(w http.ResponseWriter, r *http.Request) {
	var request callCancelRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	if request.CallID == nil {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors: map[string]string{
				"call_id": "Call id is required.",
			},
		})
		return
	}

	response, err := h.nexusClient.CallCancel(r.Context(), *request.CallID)
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to cancel call on upstream platform")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"canceled_money": response.CanceledMoney,
	})
}

func (h *LegacyCallAPIHandler) ControlRtp(w http.ResponseWriter, r *http.Request) {
	var request controlRtpRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	providerCode := strings.TrimSpace(request.ProviderCode)
	username := strings.ToLower(strings.TrimSpace(request.Username))
	errorsByField := map[string]string{}
	if providerCode == "" {
		errorsByField["provider_code"] = "Provider code is required."
	} else if len(providerCode) > 50 {
		errorsByField["provider_code"] = "Provider code must not exceed 50 characters."
	}
	if username == "" {
		errorsByField["username"] = "Username is required."
	} else if len(username) > 50 {
		errorsByField["username"] = "Username must not exceed 50 characters."
	}
	if request.RTP == nil {
		errorsByField["rtp"] = "RTP is required."
	} else if *request.RTP < 0 {
		errorsByField["rtp"] = "RTP must be at least 0."
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

	response, err := h.nexusClient.ControlRtp(r.Context(), providerCode, player.ExtUsername, *request.RTP)
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to control RTP on upstream platform")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"changed_rtp": response.ChangedRTP,
	})
}

func (h *LegacyCallAPIHandler) ControlUsersRtp(w http.ResponseWriter, r *http.Request) {
	var request controlUsersRtpRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	usernames := make([]string, 0, len(request.UserCodes))
	errorsByField := map[string]string{}
	if len(request.UserCodes) == 0 {
		errorsByField["user_codes"] = "User codes is required."
	}
	for _, userCode := range request.UserCodes {
		username := strings.ToLower(strings.TrimSpace(userCode))
		if username == "" {
			errorsByField["user_codes"] = "User codes must not contain empty usernames."
			continue
		}
		if len(username) > 50 {
			errorsByField["user_codes"] = "User codes must not exceed 50 characters."
			continue
		}
		usernames = append(usernames, username)
	}
	if request.RTP == nil {
		errorsByField["rtp"] = "RTP is required."
	} else if *request.RTP < 0 {
		errorsByField["rtp"] = "RTP must be at least 0."
	}
	if len(errorsByField) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors:  errorsByField,
		})
		return
	}

	toko, ok := auth.CurrentToko(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, authErrorResponse{Message: "Forbidden"})
		return
	}

	usernameMap, err := h.playerService.UsernameMapForToko(r.Context(), toko.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to load player mapping"})
		return
	}

	localToExternal := reverseUsernameMap(usernameMap)
	externalUsernames := make([]string, 0, len(usernames))
	for _, username := range usernames {
		extUsername, exists := localToExternal[username]
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"success": false,
				"message": "Player not found",
			})
			return
		}

		externalUsernames = append(externalUsernames, extUsername)
	}

	response, err := h.nexusClient.ControlUsersRtp(r.Context(), externalUsernames, *request.RTP)
	if err != nil || response.Status != 1 {
		writePlayerAPIUpstreamError(w, err, "Failed to control users RTP on upstream platform")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"changed_rtp": response.ChangedRTP,
	})
}

func (h *LegacyCallAPIHandler) playerForCurrentToko(w http.ResponseWriter, r *http.Request, username string) (*players.Player, bool) {
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

func mapCallPlayerRecords(records []map[string]any, usernameMap map[string]string) []map[string]any {
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
			"username":      username,
			"provider_code": record["provider_code"],
			"game_code":     record["game_code"],
			"bet":           record["bet"],
			"balance":       record["balance"],
			"total_debit":   record["total_debit"],
			"total_credit":  record["total_credit"],
			"target_rtp":    record["target_rtp"],
			"real_rtp":      record["real_rtp"],
		})
	}

	return mapped
}

func mapCallHistoryRecords(records []map[string]any, usernameMap map[string]string) []map[string]any {
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
			"id":            record["id"],
			"username":      username,
			"provider_code": record["provider_code"],
			"game_code":     record["game_code"],
			"bet":           record["bet"],
			"user_prev":     record["user_prev"],
			"user_after":    record["user_after"],
			"agent_prev":    record["agent_prev"],
			"agent_after":   record["agent_after"],
			"expect":        record["expect"],
			"missed":        record["missed"],
			"real":          record["real"],
			"rtp":           record["rtp"],
			"type":          record["type"],
			"status":        record["status"],
			"created_at":    record["created_at"],
			"updated_at":    record["updated_at"],
		})
	}

	return mapped
}

func reverseUsernameMap(usernameMap map[string]string) map[string]string {
	reversed := make(map[string]string, len(usernameMap))
	for extUsername, username := range usernameMap {
		reversed[username] = extUsername
	}

	return reversed
}
