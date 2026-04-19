package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/callmanagement"
)

type backofficeCallManagementService interface {
	Bootstrap(ctx context.Context, actor auth.PublicUser) (*callmanagement.BootstrapResult, error)
	ActivePlayers(ctx context.Context, actor auth.PublicUser) ([]callmanagement.ActivePlayerRecord, error)
	CallList(ctx context.Context, providerCode string, gameCode string) ([]callmanagement.CallOption, error)
	Apply(ctx context.Context, actor auth.PublicUser, input callmanagement.ApplyInput) (any, error)
	History(ctx context.Context, actor auth.PublicUser, offset int, limit int) ([]callmanagement.CallHistoryRecord, error)
	Cancel(ctx context.Context, callID int) (any, error)
	ControlRTP(ctx context.Context, actor auth.PublicUser, input callmanagement.ControlRTPInput) (any, error)
	ControlUsersRTP(ctx context.Context, actor auth.PublicUser, rtp float64) (any, error)
}

type BackofficeCallManagementHandler struct {
	service backofficeCallManagementService
}

type callListPayload struct {
	ProviderCode string `json:"providerCode"`
	GameCode     string `json:"gameCode"`
}

type callApplyPayload struct {
	PlayerID      int64  `json:"playerId"`
	ProviderCode  string `json:"providerCode"`
	GameCode      string `json:"gameCode"`
	CallRTP       int    `json:"callRtp"`
	CallTypeValue int    `json:"callTypeValue"`
}

type callHistoryPayload struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type callCancelPayload struct {
	CallID int `json:"callId"`
}

type controlRTPPayload struct {
	PlayerID     int64   `json:"playerId"`
	ProviderCode string  `json:"providerCode"`
	RTP          float64 `json:"rtp"`
}

type controlUsersRTPPayload struct {
	RTP float64 `json:"rtp"`
}

func NewBackofficeCallManagementHandler(service backofficeCallManagementService) *BackofficeCallManagementHandler {
	return &BackofficeCallManagementHandler{service: service}
}

func (h *BackofficeCallManagementHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	result, err := h.service.Bootstrap(r.Context(), actor)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load call management bootstrap"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

func (h *BackofficeCallManagementHandler) ActivePlayers(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	records, err := h.service.ActivePlayers(r.Context(), actor)
	if err != nil {
		writeBackofficeCallManagementError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": records,
	})
}

func (h *BackofficeCallManagementHandler) CallList(w http.ResponseWriter, r *http.Request) {
	var payload callListPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := map[string]string{}
	if strings.TrimSpace(payload.ProviderCode) == "" {
		fieldErrors["providerCode"] = "Provider wajib dipilih."
	}
	if strings.TrimSpace(payload.GameCode) == "" {
		fieldErrors["gameCode"] = "Game wajib dipilih."
	}
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	result, err := h.service.CallList(r.Context(), payload.ProviderCode, payload.GameCode)
	if err != nil {
		writeBackofficeCallManagementError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

func (h *BackofficeCallManagementHandler) Apply(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload callApplyPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := map[string]string{}
	if payload.PlayerID <= 0 {
		fieldErrors["playerId"] = "Player wajib dipilih."
	}
	if strings.TrimSpace(payload.ProviderCode) == "" {
		fieldErrors["providerCode"] = "Provider wajib dipilih."
	}
	if strings.TrimSpace(payload.GameCode) == "" {
		fieldErrors["gameCode"] = "Game wajib dipilih."
	}
	if payload.CallTypeValue != 1 && payload.CallTypeValue != 2 {
		fieldErrors["callTypeValue"] = "Call type tidak valid."
	}
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	calledMoney, err := h.service.Apply(r.Context(), actor, callmanagement.ApplyInput{
		PlayerID:      payload.PlayerID,
		ProviderCode:  payload.ProviderCode,
		GameCode:      payload.GameCode,
		CallRTP:       payload.CallRTP,
		CallTypeValue: payload.CallTypeValue,
	})
	if err != nil {
		writeBackofficeCallManagementError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Call applied",
		"data": map[string]any{
			"calledMoney": calledMoney,
		},
	})
}

func (h *BackofficeCallManagementHandler) History(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	payload := callHistoryPayload{}
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
			return
		}
	}

	records, err := h.service.History(r.Context(), actor, payload.Offset, payload.Limit)
	if err != nil {
		writeBackofficeCallManagementError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": records,
	})
}

func (h *BackofficeCallManagementHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	var payload callCancelPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	if payload.CallID <= 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors": map[string]string{
				"callId": "Call id wajib diisi.",
			},
		})
		return
	}

	canceledMoney, err := h.service.Cancel(r.Context(), payload.CallID)
	if err != nil {
		writeBackofficeCallManagementError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Call canceled",
		"data": map[string]any{
			"canceledMoney": canceledMoney,
		},
	})
}

func (h *BackofficeCallManagementHandler) ControlRTP(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload controlRTPPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := map[string]string{}
	if payload.PlayerID <= 0 {
		fieldErrors["playerId"] = "Player wajib dipilih."
	}
	if strings.TrimSpace(payload.ProviderCode) == "" {
		fieldErrors["providerCode"] = "Provider wajib dipilih."
	}
	if payload.RTP < 0 {
		fieldErrors["rtp"] = "RTP minimal 0."
	} else if payload.RTP > 95 {
		fieldErrors["rtp"] = "RTP maksimal 95."
	}
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	changedRTP, err := h.service.ControlRTP(r.Context(), actor, callmanagement.ControlRTPInput{
		PlayerID:     payload.PlayerID,
		ProviderCode: payload.ProviderCode,
		RTP:          payload.RTP,
	})
	if err != nil {
		writeBackofficeCallManagementError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "RTP updated",
		"data": map[string]any{
			"changedRtp": changedRTP,
		},
	})
}

func (h *BackofficeCallManagementHandler) ControlUsersRTP(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload controlUsersRTPPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	if payload.RTP < 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors": map[string]string{
				"rtp": "RTP minimal 0.",
			},
		})
		return
	}
	if payload.RTP > 95 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors": map[string]string{
				"rtp": "RTP maksimal 95.",
			},
		})
		return
	}

	changedRTP, err := h.service.ControlUsersRTP(r.Context(), actor, payload.RTP)
	if err != nil {
		writeBackofficeCallManagementError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Users RTP updated",
		"data": map[string]any{
			"changedRtp": changedRTP,
		},
	})
}

func writeBackofficeCallManagementError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	response := map[string]any{
		"message": "Failed to process call management request",
	}

	switch {
	case errors.Is(err, callmanagement.ErrPlayerNotFound):
		status = http.StatusNotFound
		response["message"] = "Player not found"
	case errors.Is(err, callmanagement.ErrNoAccessiblePlayers):
		status = http.StatusUnprocessableEntity
		response["message"] = "Tidak ada player yang bisa dikelola."
	case errors.Is(err, callmanagement.ErrCallListUnavailable):
		status = http.StatusBadGateway
		response["message"] = "Call list tidak tersedia dari upstream platform."
	case errors.Is(err, callmanagement.ErrCallHistoryUnavailable):
		status = http.StatusBadGateway
		response["message"] = "Call history tidak tersedia dari upstream platform."
	case errors.Is(err, callmanagement.ErrUpstreamCallFailed):
		status = http.StatusBadGateway
		response["message"] = "Aksi call gagal di upstream platform."
	case errors.Is(err, callmanagement.ErrCallCancelFailed):
		status = http.StatusBadGateway
		response["message"] = "Cancel call gagal di upstream platform."
	case errors.Is(err, callmanagement.ErrControlRTPFailed):
		status = http.StatusBadGateway
		response["message"] = "RTP control gagal di upstream platform."
	}

	writeJSON(w, status, response)
}
