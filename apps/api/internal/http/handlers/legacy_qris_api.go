package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	qrismodule "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/qris"
)

type qrisService interface {
	Generate(ctx context.Context, toko auth.Toko, params qrismodule.GenerateParams) (*qrismodule.GenerateResult, error)
	CheckStatus(ctx context.Context, toko auth.Toko, trxID string) (*qrismodule.CheckStatusResult, error)
}

type LegacyQrisAPIHandler struct {
	service qrisService
}

type generateQrisRequest struct {
	Username  string  `json:"username"`
	Amount    int64   `json:"amount"`
	Expire    *int    `json:"expire"`
	CustomRef *string `json:"custom_ref"`
}

type checkStatusRequest struct {
	TrxID string `json:"trx_id"`
}

func NewLegacyQrisAPIHandler(service qrisService) *LegacyQrisAPIHandler {
	return &LegacyQrisAPIHandler{service: service}
}

func (h *LegacyQrisAPIHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var request generateQrisRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	username := strings.TrimSpace(request.Username)
	errorsByField := map[string]string{}
	if username == "" {
		errorsByField["username"] = "Username is required."
	} else if len(username) > 255 {
		errorsByField["username"] = "Username must not exceed 255 characters."
	}
	if request.Amount < 10000 {
		errorsByField["amount"] = "Amount must be at least 10000."
	}
	if request.Expire != nil && *request.Expire < 1 {
		errorsByField["expire"] = "Expire must be at least 1."
	}
	if request.CustomRef != nil && len(strings.TrimSpace(*request.CustomRef)) > 36 {
		errorsByField["custom_ref"] = "Custom ref must not exceed 36 characters."
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

	result, err := h.service.Generate(r.Context(), toko, qrismodule.GenerateParams{
		Username:  username,
		Amount:    request.Amount,
		Expire:    request.Expire,
		CustomRef: request.CustomRef,
	})
	if err != nil {
		if errors.Is(err, qrismodule.ErrGenerateUpstreamFailure) {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "Failed to generate QRIS from upstream provider",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    result.Data,
		"trx_id":  result.TrxID,
	})
}

func (h *LegacyQrisAPIHandler) CheckStatus(w http.ResponseWriter, r *http.Request) {
	var request checkStatusRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	trxID := strings.TrimSpace(request.TrxID)
	if trxID == "" {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors: map[string]string{
				"trx_id": "Trx ID is required.",
			},
		})
		return
	}
	if len(trxID) > 255 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors: map[string]string{
				"trx_id": "Trx ID must not exceed 255 characters.",
			},
		})
		return
	}

	toko, ok := auth.CurrentToko(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, authErrorResponse{Message: "Forbidden"})
		return
	}

	result, err := h.service.CheckStatus(r.Context(), toko, trxID)
	if err != nil {
		switch {
		case errors.Is(err, qrismodule.ErrTransactionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]any{
				"success": false,
				"message": "Transaction not found",
			})
		case errors.Is(err, qrismodule.ErrCheckStatusUpstreamFailure):
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "Failed to get QRIS transaction status from upstream provider",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"trx_id":  result.TrxID,
		"status":  result.Status,
	})
}
