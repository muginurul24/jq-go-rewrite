package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/nexusplayers"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
)

type playerMutationService interface {
	CreateUser(ctx context.Context, toko auth.Toko, username string) (*nexusplayers.CreateUserResult, error)
	Deposit(ctx context.Context, toko auth.Toko, username string, amount int64, agentSign *string) (*nexusplayers.MutationResult, error)
	Withdraw(ctx context.Context, toko auth.Toko, username string, amount int64, agentSign *string) (*nexusplayers.MutationResult, error)
	WithdrawReset(ctx context.Context, toko auth.Toko, username *string, allUsers bool) (*nexusplayers.WithdrawResetResult, error)
	TransferStatus(ctx context.Context, toko auth.Toko, username string, agentSign string) (*nexusplayers.TransferStatusResult, error)
}

type LegacyPlayerMutationAPIHandler struct {
	service playerMutationService
}

type userCreateRequest struct {
	Username string `json:"username"`
}

type userBalanceMutationRequest struct {
	Username  string  `json:"username"`
	Amount    float64 `json:"amount"`
	AgentSign *string `json:"agent_sign"`
}

type userWithdrawResetRequest struct {
	Username *string `json:"username"`
	AllUsers *bool   `json:"all_users"`
}

type transferStatusRequest struct {
	Username  string `json:"username"`
	AgentSign string `json:"agent_sign"`
}

func NewLegacyPlayerMutationAPIHandler(service playerMutationService) *LegacyPlayerMutationAPIHandler {
	return &LegacyPlayerMutationAPIHandler{service: service}
}

func (h *LegacyPlayerMutationAPIHandler) UserCreate(w http.ResponseWriter, r *http.Request) {
	var request userCreateRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	username := strings.ToLower(strings.TrimSpace(request.Username))
	if username == "" {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors: map[string]string{
				"username": "Username is required.",
			},
		})
		return
	}
	if len(username) > 50 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors: map[string]string{
				"username": "Username must not exceed 50 characters.",
			},
		})
		return
	}

	toko, ok := auth.CurrentToko(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, authErrorResponse{Message: "Forbidden"})
		return
	}

	result, err := h.service.CreateUser(r.Context(), toko, username)
	if err != nil {
		switch {
		case errors.Is(err, nexusplayers.ErrDuplicateUsername):
			writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
				Message: "Validation failed",
				Errors: map[string]string{
					"username": "Username has already been taken.",
				},
			})
		case errors.Is(err, nexusplayers.ErrCreateUserUpstreamFailure):
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "Failed to create user on upstream platform",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to create user"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"username": result.Username,
	})
}

func (h *LegacyPlayerMutationAPIHandler) UserDeposit(w http.ResponseWriter, r *http.Request) {
	h.handleBalanceMutation(w, r, "deposit")
}

func (h *LegacyPlayerMutationAPIHandler) UserWithdraw(w http.ResponseWriter, r *http.Request) {
	h.handleBalanceMutation(w, r, "withdraw")
}

func (h *LegacyPlayerMutationAPIHandler) UserWithdrawReset(w http.ResponseWriter, r *http.Request) {
	var request userWithdrawResetRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	allUsers := request.AllUsers != nil && *request.AllUsers
	var username *string
	if request.Username != nil && strings.TrimSpace(*request.Username) != "" {
		normalized := strings.ToLower(strings.TrimSpace(*request.Username))
		username = &normalized
	}

	errorsByField := map[string]string{}
	if !allUsers && username == nil {
		errorsByField["username"] = "Username is required."
	}
	if username != nil && len(*username) > 50 {
		errorsByField["username"] = "Username must not exceed 50 characters."
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

	result, err := h.service.WithdrawReset(r.Context(), toko, username, allUsers)
	if err != nil {
		switch {
		case errors.Is(err, players.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]any{
				"success": false,
				"message": "Player not found",
			})
		case errors.Is(err, nexusplayers.ErrWithdrawResetUpstreamFailure):
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "Failed to reset withdraw on upstream platform",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Internal server error"})
		}
		return
	}

	payload := map[string]any{
		"success": true,
		"agent": map[string]any{
			"code":    toko.Name,
			"balance": result.AgentBalance,
		},
	}
	if result.User != nil {
		payload["user"] = map[string]any{
			"username":        result.User.Username,
			"withdraw_amount": result.User.WithdrawAmount,
			"balance":         result.User.Balance,
		}
	}
	if len(result.UserList) > 0 {
		userList := make([]map[string]any, 0, len(result.UserList))
		for _, user := range result.UserList {
			userList = append(userList, map[string]any{
				"username":        user.Username,
				"withdraw_amount": user.WithdrawAmount,
				"balance":         user.Balance,
			})
		}
		payload["user_list"] = userList
	}

	writeJSON(w, http.StatusOK, payload)
}

func (h *LegacyPlayerMutationAPIHandler) TransferStatus(w http.ResponseWriter, r *http.Request) {
	var request transferStatusRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	username := strings.ToLower(strings.TrimSpace(request.Username))
	agentSign := strings.TrimSpace(request.AgentSign)

	errorsByField := map[string]string{}
	if username == "" {
		errorsByField["username"] = "Username is required."
	} else if len(username) > 50 {
		errorsByField["username"] = "Username must not exceed 50 characters."
	}
	if agentSign == "" {
		errorsByField["agent_sign"] = "Agent sign is required."
	} else if len(agentSign) > 255 {
		errorsByField["agent_sign"] = "Agent sign must not exceed 255 characters."
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

	result, err := h.service.TransferStatus(r.Context(), toko, username, agentSign)
	if err != nil {
		switch {
		case errors.Is(err, players.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]any{
				"success": false,
				"message": "Player not found",
			})
		case errors.Is(err, nexusplayers.ErrTransferStatusUpstreamFailure):
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "Failed to get transfer status from upstream platform",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"amount":  result.Amount,
		"type":    result.Type,
		"agent": map[string]any{
			"code":    toko.Name,
			"balance": result.AgentBalance,
		},
		"user": map[string]any{
			"username": result.Username,
			"balance":  result.UserBalance,
		},
	})
}

func (h *LegacyPlayerMutationAPIHandler) handleBalanceMutation(w http.ResponseWriter, r *http.Request, mutationType string) {
	var request userBalanceMutationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	username := strings.ToLower(strings.TrimSpace(request.Username))
	errorsByField := map[string]string{}
	if username == "" {
		errorsByField["username"] = "Username is required."
	} else if len(username) > 50 {
		errorsByField["username"] = "Username must not exceed 50 characters."
	}
	if request.AgentSign != nil && len(strings.TrimSpace(*request.AgentSign)) > 255 {
		errorsByField["agent_sign"] = "Agent sign must not exceed 255 characters."
	}

	amount := int64(request.Amount)
	switch mutationType {
	case "deposit":
		if request.Amount < 10000 {
			errorsByField["amount"] = "Amount must be at least 10000."
		}
	case "withdraw":
		if request.Amount < 1 {
			errorsByField["amount"] = "Amount must be at least 1."
		}
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

	var (
		result *nexusplayers.MutationResult
		err    error
	)
	switch mutationType {
	case "deposit":
		result, err = h.service.Deposit(r.Context(), toko, username, amount, request.AgentSign)
	default:
		result, err = h.service.Withdraw(r.Context(), toko, username, amount, request.AgentSign)
	}

	if err != nil {
		writeMutationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"agent": map[string]any{
			"code":    toko.Name,
			"balance": result.AgentBalance,
		},
		"user": map[string]any{
			"username": result.Username,
			"balance":  result.UserBalance,
		},
	})
}

func writeMutationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, players.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{
			"success": false,
			"message": "Player not found",
		})
	case errors.Is(err, nexusplayers.ErrInsufficientNexusBalance):
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"message": "Insufficient balance",
		})
	case errors.Is(err, nexusplayers.ErrUpstreamUserInsufficientFunds):
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"message": "User has insufficient balance on upstream platform",
		})
	case errors.Is(err, nexusplayers.ErrWithdrawMoneyInfoUpstream):
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "Failed to get user balance from upstream platform",
		})
	case errors.Is(err, nexusplayers.ErrDepositUpstreamFailure):
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "Failed to deposit user on upstream platform",
		})
	case errors.Is(err, nexusplayers.ErrWithdrawUpstreamFailure):
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "Failed to withdraw user on upstream platform",
		})
	default:
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Internal server error"})
	}
}
