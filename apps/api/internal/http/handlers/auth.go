package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

type AuthHandler struct {
	sessionManager *scs.SessionManager
	service        *auth.Service
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type registerRequest struct {
	Username             string `json:"username"`
	Name                 string `json:"name"`
	Email                string `json:"email"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

type authBootstrapResponse struct {
	CSRFToken  string           `json:"csrfToken"`
	User       *auth.PublicUser `json:"user"`
	MFAPending bool             `json:"mfaPending,omitempty"`
}

type authSuccessResponse struct {
	CSRFToken   string           `json:"csrfToken,omitempty"`
	User        *auth.PublicUser `json:"user,omitempty"`
	RequiresMFA bool             `json:"requiresMfa,omitempty"`
}

type authErrorResponse struct {
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

type logoutSuccessResponse struct {
	Message   string `json:"message"`
	CSRFToken string `json:"csrfToken"`
}

type mfaChallengeRequest struct {
	Code string `json:"code"`
}

type mfaStatusResponse struct {
	Enabled      bool   `json:"enabled"`
	PendingSetup bool   `json:"pendingSetup"`
	Secret       string `json:"secret,omitempty"`
	OTPAuthURL   string `json:"otpauthUrl,omitempty"`
}

type mfaSetupResponse struct {
	Enabled      bool   `json:"enabled"`
	PendingSetup bool   `json:"pendingSetup"`
	Secret       string `json:"secret,omitempty"`
	OTPAuthURL   string `json:"otpauthUrl,omitempty"`
}

type mfaConfirmResponse struct {
	Enabled       bool     `json:"enabled"`
	RecoveryCodes []string `json:"recoveryCodes,omitempty"`
}

func NewAuthHandler(sessionManager *scs.SessionManager, service *auth.Service) *AuthHandler {
	return &AuthHandler{
		sessionManager: sessionManager,
		service:        service,
	}
}

func (h *AuthHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	var currentUser *auth.PublicUser
	mfaPending := h.sessionManager.GetInt64(r.Context(), auth.SessionMFAPendingUserIDKey) > 0

	if userID := h.sessionManager.GetInt64(r.Context(), auth.SessionUserIDKey); userID > 0 {
		user, err := h.service.FindUserByID(r.Context(), userID)
		if err == nil {
			currentUser = user
		}
	}

	writeJSON(w, http.StatusOK, authBootstrapResponse{
		CSRFToken:  csrf.Token(r),
		User:       currentUser,
		MFAPending: currentUser == nil && mfaPending,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	login := strings.TrimSpace(request.Login)
	if !auth.ValidLoginIdentifier(login) {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors: map[string]string{
				"login": "Username or Email is invalid.",
			},
		})
		return
	}

	user, err := h.service.AuthenticateCredentials(r.Context(), login, request.Password)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Login failed",
			Errors: map[string]string{
				"login": "These credentials do not match our records.",
			},
		})
		return
	}

	if err := h.sessionManager.RenewToken(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to renew session"})
		return
	}

	clearMFASession(h.sessionManager, r.Context())

	if user.MFAEnabled() {
		h.sessionManager.Put(r.Context(), auth.SessionMFAPendingUserIDKey, user.ID)
		h.sessionManager.Put(r.Context(), auth.SessionMFAPendingRemember, request.Remember)

		writeJSON(w, http.StatusOK, authSuccessResponse{
			CSRFToken:   csrf.Token(r),
			RequiresMFA: true,
		})
		return
	}

	h.sessionManager.Put(r.Context(), auth.SessionUserIDKey, user.ID)
	h.sessionManager.RememberMe(r.Context(), request.Remember)

	writeJSON(w, http.StatusOK, authSuccessResponse{
		CSRFToken: csrf.Token(r),
		User:      userPublicPtr(user.Public()),
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request registerRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	errorsByField := validateRegisterRequest(request)
	if len(errorsByField) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors:  errorsByField,
		})
		return
	}

	user, err := h.service.RegisterUser(r.Context(), auth.RegisterInput{
		Username: strings.TrimSpace(request.Username),
		Name:     strings.TrimSpace(request.Name),
		Email:    strings.TrimSpace(request.Email),
		Password: request.Password,
	})
	if err != nil {
		status := http.StatusInternalServerError
		response := authErrorResponse{Message: "Failed to register user"}

		if errors.Is(err, auth.ErrDuplicateUser) {
			status = http.StatusUnprocessableEntity
			response = authErrorResponse{
				Message: "Validation failed",
				Errors: map[string]string{
					"username": "Username or email has already been taken.",
				},
			}
		}

		writeJSON(w, status, response)
		return
	}

	if err := h.sessionManager.RenewToken(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to renew session"})
		return
	}

	h.sessionManager.Put(r.Context(), auth.SessionUserIDKey, user.ID)
	h.sessionManager.RememberMe(r.Context(), false)

	writeJSON(w, http.StatusCreated, authSuccessResponse{
		CSRFToken: csrf.Token(r),
		User:      user,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, authErrorResponse{Message: "Unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, authSuccessResponse{
		CSRFToken: csrf.Token(r),
		User:      &user,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	clearMFASession(h.sessionManager, r.Context())

	if err := h.sessionManager.Destroy(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to destroy session"})
		return
	}

	writeJSON(w, http.StatusOK, logoutSuccessResponse{
		Message:   "Logged out",
		CSRFToken: csrf.Token(r),
	})
}

func (h *AuthHandler) VerifyLoginMFA(w http.ResponseWriter, r *http.Request) {
	var request mfaChallengeRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	pendingUserID := h.sessionManager.GetInt64(r.Context(), auth.SessionMFAPendingUserIDKey)
	if pendingUserID == 0 {
		writeJSON(w, http.StatusUnauthorized, authErrorResponse{Message: "MFA login challenge not found"})
		return
	}

	user, err := h.service.VerifyLoginMFA(r.Context(), pendingUserID, request.Code)
	if err != nil {
		status := http.StatusUnprocessableEntity
		response := authErrorResponse{Message: "Invalid MFA code"}
		if errors.Is(err, auth.ErrInactiveUser) {
			status = http.StatusUnauthorized
			response = authErrorResponse{Message: "Unauthorized"}
		}
		writeJSON(w, status, response)
		return
	}

	remember := h.sessionManager.GetBool(r.Context(), auth.SessionMFAPendingRemember)
	clearMFASession(h.sessionManager, r.Context())
	h.sessionManager.Put(r.Context(), auth.SessionUserIDKey, user.ID)
	h.sessionManager.RememberMe(r.Context(), remember)

	writeJSON(w, http.StatusOK, authSuccessResponse{
		CSRFToken: csrf.Token(r),
		User:      user,
	})
}

func (h *AuthHandler) MFAStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, authErrorResponse{Message: "Unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, mfaStatusResponse{
		Enabled:      user.MFAEnabled,
		PendingSetup: strings.TrimSpace(h.sessionManager.GetString(r.Context(), auth.SessionMFASetupSecretKey)) != "",
		Secret:       strings.TrimSpace(h.sessionManager.GetString(r.Context(), auth.SessionMFASetupSecretKey)),
		OTPAuthURL: func() string {
			secret := strings.TrimSpace(h.sessionManager.GetString(r.Context(), auth.SessionMFASetupSecretKey))
			if secret == "" {
				return ""
			}
			return auth.BuildOTPAuthURL(user, secret)
		}(),
	})
}

func (h *AuthHandler) BeginMFASetup(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, authErrorResponse{Message: "Unauthorized"})
		return
	}

	if user.MFAEnabled {
		writeJSON(w, http.StatusConflict, authErrorResponse{Message: "MFA already enabled"})
		return
	}

	setup, err := h.service.BeginMFASetup(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to initialize MFA"})
		return
	}

	recoveryPayload, err := json.Marshal(setup.RecoveryCodes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to initialize MFA"})
		return
	}

	h.sessionManager.Put(r.Context(), auth.SessionMFASetupSecretKey, setup.Secret)
	h.sessionManager.Put(r.Context(), auth.SessionMFASetupCodesKey, string(recoveryPayload))

	writeJSON(w, http.StatusOK, mfaSetupResponse{
		Enabled:      false,
		PendingSetup: true,
		Secret:       setup.Secret,
		OTPAuthURL:   setup.OTPAuthURL,
	})
}

func (h *AuthHandler) ConfirmMFASetup(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, authErrorResponse{Message: "Unauthorized"})
		return
	}

	var request mfaChallengeRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	secret := strings.TrimSpace(h.sessionManager.GetString(r.Context(), auth.SessionMFASetupSecretKey))
	recoveryPayload := strings.TrimSpace(h.sessionManager.GetString(r.Context(), auth.SessionMFASetupCodesKey))
	if secret == "" || recoveryPayload == "" {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{Message: "MFA setup is not pending"})
		return
	}

	var recoveryCodes []string
	if err := json.Unmarshal([]byte(recoveryPayload), &recoveryCodes); err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to load recovery codes"})
		return
	}

	if err := h.service.ConfirmMFASetup(r.Context(), user.ID, secret, request.Code, recoveryCodes); err != nil {
		status := http.StatusInternalServerError
		message := "Failed to enable MFA"
		if errors.Is(err, auth.ErrMFAInvalidCode) {
			status = http.StatusUnprocessableEntity
			message = "Invalid MFA code"
		}
		writeJSON(w, status, authErrorResponse{Message: message})
		return
	}

	h.sessionManager.Remove(r.Context(), auth.SessionMFASetupSecretKey)
	h.sessionManager.Remove(r.Context(), auth.SessionMFASetupCodesKey)

	writeJSON(w, http.StatusOK, mfaConfirmResponse{
		Enabled:       true,
		RecoveryCodes: recoveryCodes,
	})
}

func (h *AuthHandler) DisableMFA(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, authErrorResponse{Message: "Unauthorized"})
		return
	}

	var request mfaChallengeRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	if err := h.service.DisableMFA(r.Context(), user.ID, request.Code); err != nil {
		status := http.StatusInternalServerError
		message := "Failed to disable MFA"
		if errors.Is(err, auth.ErrMFAInvalidCode) || errors.Is(err, auth.ErrMFANotEnabled) {
			status = http.StatusUnprocessableEntity
			message = "Invalid MFA code"
		}
		writeJSON(w, status, authErrorResponse{Message: message})
		return
	}

	writeJSON(w, http.StatusOK, mfaStatusResponse{
		Enabled:      false,
		PendingSetup: false,
	})
}

func decodeJSON(r *http.Request, target any) error {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte("[]")) {
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func validateRegisterRequest(request registerRequest) map[string]string {
	errorsByField := map[string]string{}

	username := strings.TrimSpace(request.Username)
	switch {
	case username == "":
		errorsByField["username"] = "Username is required."
	case len(username) < 5:
		errorsByField["username"] = "Username must be at least 5 characters."
	case len(username) > 20:
		errorsByField["username"] = "Username must not exceed 20 characters."
	case !usernamePattern.MatchString(username):
		errorsByField["username"] = "Username must be alphanumeric."
	}

	name := strings.TrimSpace(request.Name)
	switch {
	case name == "":
		errorsByField["name"] = "Name is required."
	case len(name) < 5:
		errorsByField["name"] = "Name must be at least 5 characters."
	case len(name) > 100:
		errorsByField["name"] = "Name must not exceed 100 characters."
	}

	email := strings.TrimSpace(request.Email)
	if email == "" || !strings.Contains(email, "@") {
		errorsByField["email"] = "A valid email address is required."
	}

	switch {
	case request.Password == "":
		errorsByField["password"] = "Password is required."
	case len(request.Password) < 8:
		errorsByField["password"] = "Password must be at least 8 characters."
	case request.Password != request.PasswordConfirmation:
		errorsByField["password_confirmation"] = "Password confirmation does not match."
	}

	return errorsByField
}

func clearMFASession(sessionManager *scs.SessionManager, ctx context.Context) {
	sessionManager.Remove(ctx, auth.SessionMFAPendingUserIDKey)
	sessionManager.Remove(ctx, auth.SessionMFAPendingRemember)
	sessionManager.Remove(ctx, auth.SessionMFASetupSecretKey)
	sessionManager.Remove(ctx, auth.SessionMFASetupCodesKey)
}

func userPublicPtr(user auth.PublicUser) *auth.PublicUser {
	return &user
}
