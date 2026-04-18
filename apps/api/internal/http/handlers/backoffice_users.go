package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/users"
)

type backofficeUsersService interface {
	ListForBackoffice(ctx context.Context, actor auth.PublicUser, input users.AdminListInput) (*users.AdminListResult, error)
	FindDetailForBackoffice(ctx context.Context, actor auth.PublicUser, userID int64) (*users.AdminUserRecord, error)
	CreateForBackoffice(ctx context.Context, actor auth.PublicUser, input users.CreateInput) (*users.AdminUserRecord, error)
	UpdateForBackoffice(ctx context.Context, actor auth.PublicUser, userID int64, input users.UpdateInput) (*users.AdminUserRecord, error)
}

type BackofficeUsersHandler struct {
	service backofficeUsersService
}

type backofficeUserPayload struct {
	Username string  `json:"username"`
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Role     string  `json:"role"`
	IsActive bool    `json:"isActive"`
	Password *string `json:"password,omitempty"`
}

func NewBackofficeUsersHandler(service backofficeUsersService) *BackofficeUsersHandler {
	return &BackofficeUsersHandler{service: service}
}

func (h *BackofficeUsersHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	input, err := parseBackofficeUsersListInput(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}

	result, err := h.service.ListForBackoffice(r.Context(), actor, input)
	if err != nil {
		writeUserServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": presentUsers(result.Data),
		"meta": map[string]any{
			"page":       result.Page,
			"perPage":    result.PerPage,
			"total":      result.Total,
			"totalPages": result.TotalPages,
		},
		"summary": map[string]any{
			"totalUsers":  result.Summary.TotalUsers,
			"activeUsers": result.Summary.ActiveUsers,
			"adminUsers":  result.Summary.AdminUsers,
			"endUsers":    result.Summary.EndUsers,
		},
	})
}

func (h *BackofficeUsersHandler) Detail(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil || userID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid user ID"})
		return
	}

	record, err := h.service.FindDetailForBackoffice(r.Context(), actor, userID)
	if err != nil {
		writeUserServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": presentUser(*record),
	})
}

func (h *BackofficeUsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	var payload backofficeUserPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := validateBackofficeUserPayload(payload, true)
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	record, err := h.service.CreateForBackoffice(r.Context(), actor, users.CreateInput{
		Username: strings.TrimSpace(payload.Username),
		Name:     strings.TrimSpace(payload.Name),
		Email:    strings.TrimSpace(payload.Email),
		Role:     strings.TrimSpace(payload.Role),
		IsActive: payload.IsActive,
		Password: strings.TrimSpace(*payload.Password),
	})
	if err != nil {
		writeUserServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "User created",
		"data":    presentUser(*record),
	})
}

func (h *BackofficeUsersHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil || userID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid user ID"})
		return
	}

	var payload backofficeUserPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid JSON payload"})
		return
	}

	fieldErrors := validateBackofficeUserPayload(payload, false)
	if len(fieldErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation failed",
			"errors":  fieldErrors,
		})
		return
	}

	record, err := h.service.UpdateForBackoffice(r.Context(), actor, userID, users.UpdateInput{
		Username: strings.TrimSpace(payload.Username),
		Name:     strings.TrimSpace(payload.Name),
		Email:    strings.TrimSpace(payload.Email),
		Role:     strings.TrimSpace(payload.Role),
		IsActive: payload.IsActive,
		Password: payload.Password,
	})
	if err != nil {
		writeUserServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "User updated",
		"data":    presentUser(*record),
	})
}

func parseBackofficeUsersListInput(r *http.Request) (users.AdminListInput, error) {
	page, err := parsePositiveInt(r.URL.Query().Get("page"), 1)
	if err != nil {
		return users.AdminListInput{}, err
	}

	perPage, err := parsePositiveInt(r.URL.Query().Get("per_page"), 25)
	if err != nil {
		return users.AdminListInput{}, err
	}

	return users.AdminListInput{
		Search:  strings.TrimSpace(r.URL.Query().Get("search")),
		Role:    strings.TrimSpace(r.URL.Query().Get("role")),
		Status:  strings.TrimSpace(r.URL.Query().Get("status")),
		Page:    page,
		PerPage: perPage,
	}, nil
}

func presentUsers(items []users.AdminUserRecord) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, presentUser(item))
	}
	return rows
}

func presentUser(item users.AdminUserRecord) map[string]any {
	return map[string]any{
		"id":        item.ID,
		"username":  item.Username,
		"name":      item.Name,
		"email":     item.Email,
		"role":      item.Role,
		"isActive":  item.IsActive,
		"createdAt": item.CreatedAt,
		"updatedAt": item.UpdatedAt,
	}
}

func validateBackofficeUserPayload(payload backofficeUserPayload, requirePassword bool) map[string]string {
	errorsByField := map[string]string{}

	if strings.TrimSpace(payload.Username) == "" {
		errorsByField["username"] = "Username is required."
	}
	if strings.TrimSpace(payload.Name) == "" {
		errorsByField["name"] = "Name is required."
	}
	email := strings.TrimSpace(payload.Email)
	if email == "" || !strings.Contains(email, "@") {
		errorsByField["email"] = "A valid email address is required."
	}
	role := strings.TrimSpace(payload.Role)
	if role == "" {
		errorsByField["role"] = "Role is required."
	}
	if requirePassword {
		if payload.Password == nil || strings.TrimSpace(*payload.Password) == "" {
			errorsByField["password"] = "Password is required."
		} else if len(strings.TrimSpace(*payload.Password)) < 8 {
			errorsByField["password"] = "Password must be at least 8 characters."
		}
	}

	if payload.Password != nil && strings.TrimSpace(*payload.Password) != "" && len(strings.TrimSpace(*payload.Password)) < 8 {
		errorsByField["password"] = "Password must be at least 8 characters."
	}

	return errorsByField
}

func writeUserServiceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	response := map[string]any{
		"message": "Failed to persist user",
	}

	switch {
	case errors.Is(err, users.ErrForbidden):
		status = http.StatusForbidden
		response["message"] = "Forbidden"
	case errors.Is(err, users.ErrNotFound):
		status = http.StatusNotFound
		response["message"] = "User not found"
	case errors.Is(err, users.ErrInvalidRole):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"role": "Role tidak diizinkan untuk actor saat ini.",
		}
	case errors.Is(err, users.ErrDuplicateUsername):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"username": "Username has already been taken.",
		}
	case errors.Is(err, users.ErrDuplicateEmail):
		status = http.StatusUnprocessableEntity
		response["message"] = "Validation failed"
		response["errors"] = map[string]string{
			"email": "Email has already been taken.",
		}
	}

	writeJSON(w, status, response)
}
