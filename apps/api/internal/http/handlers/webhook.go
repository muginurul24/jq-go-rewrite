package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/jobs"
)

type webhookQueue interface {
	EnqueueProcessQRISCallback(ctx context.Context, payload jobs.QRISCallbackPayload) error
	EnqueueProcessDisbursementCallback(ctx context.Context, payload jobs.DisbursementCallbackPayload) error
}

type WebhookHandler struct {
	globalUUID string
	queue      webhookQueue
}

type qrisWebhookRequest struct {
	Amount     int64   `json:"amount"`
	TerminalID string  `json:"terminal_id"`
	MerchantID string  `json:"merchant_id"`
	TrxID      string  `json:"trx_id"`
	RRN        *string `json:"rrn,omitempty"`
	CustomRef  *string `json:"custom_ref,omitempty"`
	Vendor     *string `json:"vendor,omitempty"`
	Status     string  `json:"status"`
	CreatedAt  *string `json:"created_at,omitempty"`
	FinishAt   *string `json:"finish_at,omitempty"`
}

type disbursementWebhookRequest struct {
	Amount          int64   `json:"amount"`
	PartnerRefNo    string  `json:"partner_ref_no"`
	Status          string  `json:"status"`
	TransactionDate *string `json:"transaction_date,omitempty"`
	MerchantID      string  `json:"merchant_id"`
}

func NewWebhookHandler(cfg config.Config, queue webhookQueue) *WebhookHandler {
	return &WebhookHandler{
		globalUUID: strings.TrimSpace(cfg.Integrations.QRIS.GlobalUUID),
		queue:      queue,
	}
}

func (h *WebhookHandler) QRIS(w http.ResponseWriter, r *http.Request) {
	var request qrisWebhookRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	errorsByField := map[string]string{}
	if request.Amount < 1 {
		errorsByField["amount"] = "Amount must be at least 1."
	}
	if trimmed := strings.TrimSpace(request.TerminalID); trimmed == "" {
		errorsByField["terminal_id"] = "Terminal id is required."
	} else if len(trimmed) > 255 {
		errorsByField["terminal_id"] = "Terminal id must not exceed 255 characters."
	}
	if trimmed := strings.TrimSpace(request.TrxID); trimmed == "" {
		errorsByField["trx_id"] = "Transaction id is required."
	} else if len(trimmed) > 255 {
		errorsByField["trx_id"] = "Transaction id must not exceed 255 characters."
	}
	if merchantID := strings.TrimSpace(request.MerchantID); merchantID == "" {
		errorsByField["merchant_id"] = "Merchant id is required."
	} else if merchantID != h.globalUUID {
		errorsByField["merchant_id"] = "Merchant id is invalid."
	}
	if request.RRN != nil && len(strings.TrimSpace(*request.RRN)) > 255 {
		errorsByField["rrn"] = "RRN must not exceed 255 characters."
	}
	if request.CustomRef != nil && len(strings.TrimSpace(*request.CustomRef)) > 36 {
		errorsByField["custom_ref"] = "Custom ref must not exceed 36 characters."
	}
	if request.Vendor != nil && len(strings.TrimSpace(*request.Vendor)) > 255 {
		errorsByField["vendor"] = "Vendor must not exceed 255 characters."
	}
	if !validCallbackStatus(request.Status) {
		errorsByField["status"] = "Status must be one of: pending, success, failed, expired."
	}

	if len(errorsByField) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors:  errorsByField,
		})
		return
	}

	if err := h.queue.EnqueueProcessQRISCallback(r.Context(), jobs.QRISCallbackPayload{
		Amount:     request.Amount,
		TerminalID: strings.TrimSpace(request.TerminalID),
		MerchantID: strings.TrimSpace(request.MerchantID),
		TrxID:      strings.TrimSpace(request.TrxID),
		RRN:        trimOptionalPointer(request.RRN),
		CustomRef:  trimOptionalPointer(request.CustomRef),
		Vendor:     trimOptionalPointer(request.Vendor),
		Status:     strings.ToLower(strings.TrimSpace(request.Status)),
		CreatedAt:  trimOptionalPointer(request.CreatedAt),
		FinishAt:   trimOptionalPointer(request.FinishAt),
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "OK",
	})
}

func (h *WebhookHandler) Disbursement(w http.ResponseWriter, r *http.Request) {
	var request disbursementWebhookRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	errorsByField := map[string]string{}
	if request.Amount < 1 {
		errorsByField["amount"] = "Amount must be at least 1."
	}
	if trimmed := strings.TrimSpace(request.PartnerRefNo); trimmed == "" {
		errorsByField["partner_ref_no"] = "Partner ref no is required."
	} else if len(trimmed) > 255 {
		errorsByField["partner_ref_no"] = "Partner ref no must not exceed 255 characters."
	}
	if merchantID := strings.TrimSpace(request.MerchantID); merchantID == "" {
		errorsByField["merchant_id"] = "Merchant id is required."
	} else if merchantID != h.globalUUID {
		errorsByField["merchant_id"] = "Merchant id is invalid."
	}
	if !validCallbackStatus(request.Status) {
		errorsByField["status"] = "Status must be one of: pending, success, failed, expired."
	}

	if len(errorsByField) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors:  errorsByField,
		})
		return
	}

	if err := h.queue.EnqueueProcessDisbursementCallback(r.Context(), jobs.DisbursementCallbackPayload{
		Amount:          request.Amount,
		PartnerRefNo:    strings.TrimSpace(request.PartnerRefNo),
		Status:          strings.ToLower(strings.TrimSpace(request.Status)),
		TransactionDate: trimOptionalPointer(request.TransactionDate),
		MerchantID:      strings.TrimSpace(request.MerchantID),
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  true,
		"message": "OK",
	})
}

func validCallbackStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "success", "failed", "expired":
		return true
	default:
		return false
	}
}

func trimOptionalPointer(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
