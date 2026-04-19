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
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/tokos"
)

type BackofficeTokosHandler struct {
	service       backofficeTokoService
	notifications backofficeTokoNotifications
}

type tokoPayload struct {
	UserID      int64   `json:"userId"`
	Name        string  `json:"name"`
	CallbackURL *string `json:"callbackUrl"`
	IsActive    bool    `json:"isActive"`
}

type backofficeTokoService interface {
	ListForBackoffice(ctx context.Context, user auth.PublicUser, input tokos.AdminListInput) (*tokos.AdminListResult, error)
	FindDetailForBackoffice(ctx context.Context, user auth.PublicUser, tokoID int64) (*tokos.AdminTokoRecord, error)
	CreateForBackoffice(ctx context.Context, actor auth.PublicUser, input tokos.CreateInput) (*tokos.AdminTokoRecord, error)
	UpdateForBackoffice(ctx context.Context, actor auth.PublicUser, tokoID int64, input tokos.UpdateInput) (*tokos.AdminTokoRecord, error)
	RegenerateTokenForBackoffice(ctx context.Context, actor auth.PublicUser, tokoID int64) (*tokos.AdminTokoRecord, error)
}

type backofficeTokoNotifications interface {
	NotifyTokoCreated(ctx context.Context, ownerUserID int64, tokoName string, ownerName string) error
}

func NewBackofficeTokosHandler(service backofficeTokoService) *BackofficeTokosHandler {
	return &BackofficeTokosHandler{service: service}
}

func (h *BackofficeTokosHandler) WithNotifications(service backofficeTokoNotifications) *BackofficeTokosHandler {
	h.notifications = service
	return h
}

func (h *BackofficeTokosHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	input, err := parseBackofficeTokosListInput(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}

	result, err := h.service.ListForBackoffice(r.Context(), user, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load tokos"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": presentTokos(result.Data),
		"meta": map[string]any{
			"page":       result.Page,
			"perPage":    result.PerPage,
			"total":      result.Total,
			"totalPages": result.TotalPages,
		},
		"summary": map[string]any{
			"totalTokos":    result.Summary.TotalTokos,
			"activeTokos":   result.Summary.ActiveTokos,
			"totalPending":  result.Summary.TotalPending,
			"totalSettle":   result.Summary.TotalSettle,
			"totalNexusggr": result.Summary.TotalNexusggr,
		},
		"filters": map[string]any{
			"owners": result.Owners,
		},
	})
}

func (h *BackofficeTokosHandler) Detail(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	tokoID, err := strconv.ParseInt(chi.URLParam(r, "tokoID"), 10, 64)
	if err != nil || tokoID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid toko ID"})
		return
	}

	record, err := h.service.FindDetailForBackoffice(r.Context(), user, tokoID)
	if err != nil {
		status := http.StatusInternalServerError
		message := "Failed to load toko"
		if errors.Is(err, tokos.ErrNotFound) {
			status = http.StatusNotFound
			message = "Toko not found"
		}
		writeJSON(w, status, map[string]string{"message": message})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": presentTokoDetail(*record),
	})
}

func (h *BackofficeTokosHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload tokoPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := validateTokoPayload(payload, isGlobalBackofficeRole(user.Role))
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	record, err := h.service.CreateForBackoffice(r.Context(), user, tokos.CreateInput{
		UserID:      payload.UserID,
		Name:        payload.Name,
		CallbackURL: payload.CallbackURL,
		IsActive:    payload.IsActive,
	})
	if err != nil {
		writeTokoMutationError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Toko created",
		"data":    presentTokoDetail(*record),
	})

	if h.notifications != nil {
		_ = h.notifications.NotifyTokoCreated(r.Context(), record.UserID, record.Name, record.OwnerName)
	}
}

func (h *BackofficeTokosHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	tokoID, err := strconv.ParseInt(chi.URLParam(r, "tokoID"), 10, 64)
	if err != nil || tokoID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid toko ID"})
		return
	}

	var payload tokoPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := validateTokoPayload(payload, isGlobalBackofficeRole(user.Role))
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	record, err := h.service.UpdateForBackoffice(r.Context(), user, tokoID, tokos.UpdateInput{
		UserID:      payload.UserID,
		Name:        payload.Name,
		CallbackURL: payload.CallbackURL,
		IsActive:    payload.IsActive,
	})
	if err != nil {
		writeTokoMutationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Toko updated",
		"data":    presentTokoDetail(*record),
	})
}

func (h *BackofficeTokosHandler) RegenerateToken(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	tokoID, err := strconv.ParseInt(chi.URLParam(r, "tokoID"), 10, 64)
	if err != nil || tokoID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid toko ID"})
		return
	}

	record, err := h.service.RegenerateTokenForBackoffice(r.Context(), user, tokoID)
	if err != nil {
		writeTokoMutationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Token regenerated",
		"data":    presentTokoDetail(*record),
	})
}

func parseBackofficeTokosListInput(r *http.Request) (tokos.AdminListInput, error) {
	page, err := parsePositiveInt(r.URL.Query().Get("page"), 1)
	if err != nil {
		return tokos.AdminListInput{}, fmt.Errorf("invalid page")
	}

	perPage, err := parsePositiveInt(r.URL.Query().Get("per_page"), 25)
	if err != nil {
		return tokos.AdminListInput{}, fmt.Errorf("invalid per_page")
	}

	ownerID, err := parseTokoInt64Optional(r.URL.Query().Get("owner_id"))
	if err != nil {
		return tokos.AdminListInput{}, fmt.Errorf("invalid owner_id")
	}

	return tokos.AdminListInput{
		Search:  r.URL.Query().Get("search"),
		Status:  r.URL.Query().Get("status"),
		OwnerID: ownerID,
		Page:    page,
		PerPage: perPage,
	}, nil
}

func presentTokos(records []tokos.AdminTokoRecord) []map[string]any {
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		result = append(result, presentTokoListItem(record))
	}

	return result
}

func presentTokoListItem(record tokos.AdminTokoRecord) map[string]any {
	callbackHost := ""
	if record.CallbackURL != nil {
		callbackHost = strings.TrimPrefix(strings.TrimPrefix(*record.CallbackURL, "https://"), "http://")
	}

	return map[string]any{
		"id":            record.ID,
		"userId":        record.UserID,
		"ownerUsername": record.OwnerUsername,
		"ownerName":     record.OwnerName,
		"name":          record.Name,
		"callbackUrl":   record.CallbackURL,
		"callbackHost":  callbackHost,
		"tokenPreview":  maskToken(record.Token),
		"isActive":      record.IsActive,
		"balances": map[string]any{
			"pending":  record.Pending,
			"settle":   record.Settle,
			"nexusggr": record.Nexusggr,
		},
		"createdAt": record.CreatedAt,
		"updatedAt": record.UpdatedAt,
	}
}

func presentTokoDetail(record tokos.AdminTokoRecord) map[string]any {
	payload := presentTokoListItem(record)
	payload["token"] = record.Token
	return payload
}

func validateTokoPayload(payload tokoPayload, requireOwner bool) map[string]string {
	errorsByField := map[string]string{}

	if requireOwner && payload.UserID <= 0 {
		errorsByField["userId"] = "Owner is required."
	}

	if strings.TrimSpace(payload.Name) == "" {
		errorsByField["name"] = "Nama toko wajib diisi."
	}

	if payload.CallbackURL != nil {
		trimmed := strings.TrimSpace(*payload.CallbackURL)
		if len(trimmed) > 2048 {
			errorsByField["callbackUrl"] = "Callback URL terlalu panjang."
		}
	}

	return errorsByField
}

func writeTokoMutationError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	response := map[string]any{
		"message": "Failed to persist toko",
	}

	switch {
	case errors.Is(err, tokos.ErrNotFound):
		status = http.StatusNotFound
		response["message"] = "Toko not found"
	case errors.Is(err, tokos.ErrInvalidOwner):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"userId": "Owner tidak valid.",
		}
	case errors.Is(err, tokos.ErrDuplicateCallbackURL):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"callbackUrl": "Callback URL sudah dipakai toko lain.",
		}
	}

	writeJSON(w, status, response)
}

func maskToken(token *string) *string {
	if token == nil || strings.TrimSpace(*token) == "" {
		return nil
	}

	trimmed := strings.TrimSpace(*token)
	if len(trimmed) <= 12 {
		return &trimmed
	}

	masked := trimmed[:8] + "..." + trimmed[len(trimmed)-6:]
	return &masked
}

func isGlobalBackofficeRole(role string) bool {
	return role == "dev" || role == "superadmin"
}

func parseTokoInt64Optional(raw string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}

	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid int64")
	}

	return value, nil
}
