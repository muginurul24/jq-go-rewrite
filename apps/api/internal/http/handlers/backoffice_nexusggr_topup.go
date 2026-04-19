package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/nexusggrtopup"
)

type backofficeTopupService interface {
	Bootstrap(ctx context.Context, actor auth.PublicUser, selectedTokoID *int64) (*nexusggrtopup.BootstrapResult, error)
	Generate(ctx context.Context, actor auth.PublicUser, tokoID int64, amount int64) (*nexusggrtopup.GenerateResult, error)
	CheckStatus(ctx context.Context, actor auth.PublicUser, tokoID int64, transactionCode string) (*nexusggrtopup.StatusResult, error)
}

type BackofficeNexusggrTopupHandler struct {
	service backofficeTopupService
}

type topupGeneratePayload struct {
	TokoID int64 `json:"tokoId"`
	Amount int64 `json:"amount"`
}

type topupStatusPayload struct {
	TokoID          int64  `json:"tokoId"`
	TransactionCode string `json:"transactionCode"`
}

func NewBackofficeNexusggrTopupHandler(service backofficeTopupService) *BackofficeNexusggrTopupHandler {
	return &BackofficeNexusggrTopupHandler{service: service}
}

func (h *BackofficeNexusggrTopupHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	selectedTokoID, err := parseOptionalQueryInt64(r, "toko_id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid toko_id"})
		return
	}

	result, err := h.service.Bootstrap(r.Context(), actor, selectedTokoID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load topup state"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"tokos":        result.Tokos,
			"selectedToko": result.SelectedToko,
			"topupRatio":   result.TopupRatio,
			"topupRule":    result.TopupRule,
			"pendingTopup": result.PendingTopup,
		},
	})
}

func (h *BackofficeNexusggrTopupHandler) Generate(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload topupGeneratePayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	if payload.TokoID <= 0 || payload.Amount <= 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors": map[string]string{
				"tokoId": "Toko wajib dipilih.",
				"amount": "Nominal topup wajib diisi.",
			},
		})
		return
	}

	result, err := h.service.Generate(r.Context(), actor, payload.TokoID, payload.Amount)
	if err != nil {
		writeTopupError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "QRIS generated",
		"data": map[string]any{
			"selectedToko": result.SelectedToko,
			"topupRatio":   result.TopupRatio,
			"topupRule":    result.TopupRule,
			"pendingTopup": result.PendingTopup,
		},
	})
}

func (h *BackofficeNexusggrTopupHandler) CheckStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload topupStatusPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	if payload.TokoID <= 0 || payload.TransactionCode == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors": map[string]string{
				"tokoId":          "Toko wajib dipilih.",
				"transactionCode": "Kode transaksi wajib diisi.",
			},
		})
		return
	}

	result, err := h.service.CheckStatus(r.Context(), actor, payload.TokoID, payload.TransactionCode)
	if err != nil {
		writeTopupError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

func parseOptionalQueryInt64(r *http.Request, key string) (*int64, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, err
	}

	return &value, nil
}

func writeTopupError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	response := map[string]any{
		"message": "Failed to process NexusGGR topup",
	}

	switch {
	case errors.Is(err, nexusggrtopup.ErrTokoNotFound):
		status = http.StatusNotFound
		response["message"] = "Toko not found"
	case errors.Is(err, nexusggrtopup.ErrTopupAmountTooSmall):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"amount": "Minimum topup Rp 1.000.",
		}
	case errors.Is(err, nexusggrtopup.ErrGenerateTopupFailed):
		status = http.StatusBadGateway
		response["message"] = "Gagal generate QRIS"
	case errors.Is(err, nexusggrtopup.ErrTopupTransactionNotFound):
		status = http.StatusNotFound
		response["message"] = "Pending topup transaction not found"
	}

	writeJSON(w, status, response)
}
