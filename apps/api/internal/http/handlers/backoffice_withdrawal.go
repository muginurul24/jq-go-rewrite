package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/withdrawals"
)

type backofficeWithdrawalService interface {
	Bootstrap(ctx context.Context, actor auth.PublicUser, selectedTokoID *int64) (*withdrawals.BootstrapResult, error)
	Inquiry(ctx context.Context, actor auth.PublicUser, tokoID int64, bankID int64, amount int64) (*withdrawals.InquiryResult, error)
	Submit(ctx context.Context, actor auth.PublicUser, tokoID int64, bankID int64, amount int64, inquiryID int64) (*withdrawals.SubmitResult, error)
}

type BackofficeWithdrawalHandler struct {
	service backofficeWithdrawalService
}

type withdrawalInquiryPayload struct {
	TokoID int64 `json:"tokoId"`
	BankID int64 `json:"bankId"`
	Amount int64 `json:"amount"`
}

type withdrawalSubmitPayload struct {
	TokoID    int64 `json:"tokoId"`
	BankID    int64 `json:"bankId"`
	Amount    int64 `json:"amount"`
	InquiryID int64 `json:"inquiryId"`
}

func NewBackofficeWithdrawalHandler(service backofficeWithdrawalService) *BackofficeWithdrawalHandler {
	return &BackofficeWithdrawalHandler{service: service}
}

func (h *BackofficeWithdrawalHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load withdrawal state"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"tokos":         result.Tokos,
			"selectedToko":  result.SelectedToko,
			"banks":         result.Banks,
			"feePercentage": result.FeePercentage,
			"minimumAmount": result.MinimumAmount,
		},
	})
}

func (h *BackofficeWithdrawalHandler) Inquiry(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload withdrawalInquiryPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := validateWithdrawalInput(payload.TokoID, payload.BankID, payload.Amount)
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	result, err := h.service.Inquiry(r.Context(), actor, payload.TokoID, payload.BankID, payload.Amount)
	if err != nil {
		writeWithdrawalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Rekening terverifikasi",
		"data":    result,
	})
}

func (h *BackofficeWithdrawalHandler) Submit(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload withdrawalSubmitPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := validateWithdrawalInput(payload.TokoID, payload.BankID, payload.Amount)
	if payload.InquiryID <= 0 {
		fieldErrors["inquiryId"] = "Hasil inquiry tidak valid."
	}
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	result, err := h.service.Submit(r.Context(), actor, payload.TokoID, payload.BankID, payload.Amount, payload.InquiryID)
	if err != nil {
		writeWithdrawalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Transfer berhasil",
		"data":    result,
	})
}

func validateWithdrawalInput(tokoID int64, bankID int64, amount int64) map[string]string {
	errorsByField := map[string]string{}

	if tokoID <= 0 {
		errorsByField["tokoId"] = "Toko wajib dipilih."
	}
	if bankID <= 0 {
		errorsByField["bankId"] = "Rekening tujuan wajib dipilih."
	}
	if amount <= 0 {
		errorsByField["amount"] = "Nominal withdrawal wajib diisi."
	}

	return errorsByField
}

func writeWithdrawalError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	response := map[string]any{
		"message": "Failed to process withdrawal",
	}

	switch {
	case errors.Is(err, withdrawals.ErrTokoNotFound):
		status = http.StatusNotFound
		response["message"] = "Toko not found"
	case errors.Is(err, withdrawals.ErrBankNotFound):
		status = http.StatusNotFound
		response["message"] = "Bank not found"
	case errors.Is(err, withdrawals.ErrAmountTooSmall):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"amount": "Minimum withdrawal Rp 25.000.",
		}
	case errors.Is(err, withdrawals.ErrSettleBalanceNotEnough):
		status = http.StatusUnprocessableEntity
		response["message"] = "Saldo settle tidak cukup."
	case errors.Is(err, withdrawals.ErrInquiryFailed):
		status = http.StatusBadGateway
		response["message"] = "Inquiry gagal"
	case errors.Is(err, withdrawals.ErrInquiryStateNotFound):
		status = http.StatusNotFound
		response["message"] = "Data inquiry tidak ditemukan. Silakan ulangi verifikasi rekening."
	case errors.Is(err, withdrawals.ErrTransferFailed):
		status = http.StatusBadGateway
		response["message"] = "Transfer gagal"
	}

	writeJSON(w, status, response)
}
