package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/banks"
)

type backofficeBanksService interface {
	ListForBackoffice(ctx context.Context, actor auth.PublicUser, input banks.AdminListInput) (*banks.AdminListResult, error)
	CreateForBackoffice(ctx context.Context, actor auth.PublicUser, input banks.CreateInput) (*banks.AdminBankRecord, error)
	UpdateForBackoffice(ctx context.Context, actor auth.PublicUser, bankID int64, input banks.UpdateInput) (*banks.AdminBankRecord, error)
	InquiryForBackoffice(ctx context.Context, actor auth.PublicUser, userID int64, bankCode string, accountNumber string) (*banks.InquiryResult, error)
}

type BackofficeBanksHandler struct {
	service backofficeBanksService
}

type bankPayload struct {
	UserID        int64  `json:"userId"`
	BankCode      string `json:"bankCode"`
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
}

type bankInquiryPayload struct {
	UserID        int64  `json:"userId"`
	BankCode      string `json:"bankCode"`
	AccountNumber string `json:"accountNumber"`
}

func NewBackofficeBanksHandler(service backofficeBanksService) *BackofficeBanksHandler {
	return &BackofficeBanksHandler{service: service}
}

func (h *BackofficeBanksHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	input, err := parseBackofficeBanksListInput(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}

	result, err := h.service.ListForBackoffice(r.Context(), actor, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load banks"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result.Data,
		"meta": map[string]any{
			"page":       result.Page,
			"perPage":    result.PerPage,
			"total":      result.Total,
			"totalPages": result.TotalPages,
		},
		"filters": map[string]any{
			"owners": result.Owners,
		},
	})
}

func (h *BackofficeBanksHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload bankPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := validateBankPayload(payload, isGlobalBackofficeRole(actor.Role))
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	record, err := h.service.CreateForBackoffice(r.Context(), actor, banks.CreateInput{
		UserID:        payload.UserID,
		BankCode:      payload.BankCode,
		AccountNumber: payload.AccountNumber,
		AccountName:   payload.AccountName,
	})
	if err != nil {
		writeBankMutationError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Bank created",
		"data":    record,
	})
}

func (h *BackofficeBanksHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	bankID, err := strconv.ParseInt(chi.URLParam(r, "bankID"), 10, 64)
	if err != nil || bankID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid bank ID"})
		return
	}

	var payload bankPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := validateBankPayload(payload, isGlobalBackofficeRole(actor.Role))
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	record, err := h.service.UpdateForBackoffice(r.Context(), actor, bankID, banks.UpdateInput{
		UserID:        payload.UserID,
		BankCode:      payload.BankCode,
		AccountNumber: payload.AccountNumber,
		AccountName:   payload.AccountName,
	})
	if err != nil {
		writeBankMutationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Bank updated",
		"data":    record,
	})
}

func (h *BackofficeBanksHandler) Inquiry(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload bankInquiryPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := map[string]string{}
	if isGlobalBackofficeRole(actor.Role) && payload.UserID <= 0 {
		fieldErrors["userId"] = "Owner is required."
	}
	if strings.TrimSpace(payload.BankCode) == "" {
		fieldErrors["bankCode"] = "Bank wajib dipilih."
	}
	if strings.TrimSpace(payload.AccountNumber) == "" {
		fieldErrors["accountNumber"] = "Nomor rekening wajib diisi."
	}
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	result, err := h.service.InquiryForBackoffice(r.Context(), actor, payload.UserID, payload.BankCode, payload.AccountNumber)
	if err != nil {
		writeBankMutationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Rekening ditemukan",
		"data":    result,
	})
}

func parseBackofficeBanksListInput(r *http.Request) (banks.AdminListInput, error) {
	page, err := parsePositiveInt(r.URL.Query().Get("page"), 1)
	if err != nil {
		return banks.AdminListInput{}, fmt.Errorf("invalid page")
	}

	perPage, err := parsePositiveInt(r.URL.Query().Get("per_page"), 25)
	if err != nil {
		return banks.AdminListInput{}, fmt.Errorf("invalid per_page")
	}

	ownerID, err := parseTokoInt64Optional(r.URL.Query().Get("owner_id"))
	if err != nil {
		return banks.AdminListInput{}, fmt.Errorf("invalid owner_id")
	}

	return banks.AdminListInput{
		Search:  r.URL.Query().Get("search"),
		OwnerID: ownerID,
		Page:    page,
		PerPage: perPage,
	}, nil
}

func validateBankPayload(payload bankPayload, requireOwner bool) map[string]string {
	errorsByField := map[string]string{}

	if requireOwner && payload.UserID <= 0 {
		errorsByField["userId"] = "Owner is required."
	}
	if strings.TrimSpace(payload.BankCode) == "" {
		errorsByField["bankCode"] = "Bank wajib dipilih."
	}
	if strings.TrimSpace(payload.AccountNumber) == "" {
		errorsByField["accountNumber"] = "Nomor rekening wajib diisi."
	}
	if strings.TrimSpace(payload.AccountName) == "" {
		errorsByField["accountName"] = "Nama rekening wajib diisi."
	}

	return errorsByField
}

func writeBankMutationError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	response := map[string]any{
		"message": "Failed to process bank",
	}

	switch {
	case errors.Is(err, banks.ErrNotFound):
		status = http.StatusNotFound
		response["message"] = "Bank not found"
	case errors.Is(err, banks.ErrOwnerRequired):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"userId": "Owner is required.",
		}
	case errors.Is(err, banks.ErrInvalidBankCode):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"bankCode": "Kode bank tidak valid.",
		}
	case errors.Is(err, banks.ErrDuplicateAccountNumber):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"accountNumber": "Nomor rekening sudah dipakai.",
		}
	case errors.Is(err, banks.ErrInquiryFailed):
		status = http.StatusBadGateway
		response["message"] = "Inquiry gagal"
	}

	writeJSON(w, status, response)
}
