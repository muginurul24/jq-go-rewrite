package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/nexusplayers"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
)

type fakePlayerMutationService struct {
	createResult   *nexusplayers.CreateUserResult
	mutationResult *nexusplayers.MutationResult
	err            error
	lastUsername   string
	lastAmount     int64
}

func (f *fakePlayerMutationService) CreateUser(_ context.Context, _ auth.Toko, username string) (*nexusplayers.CreateUserResult, error) {
	f.lastUsername = username
	return f.createResult, f.err
}

func (f *fakePlayerMutationService) Deposit(_ context.Context, _ auth.Toko, username string, amount int64, _ *string) (*nexusplayers.MutationResult, error) {
	f.lastUsername = username
	f.lastAmount = amount
	return f.mutationResult, f.err
}

func (f *fakePlayerMutationService) Withdraw(_ context.Context, _ auth.Toko, username string, amount int64, _ *string) (*nexusplayers.MutationResult, error) {
	f.lastUsername = username
	f.lastAmount = amount
	return f.mutationResult, f.err
}

func (f *fakePlayerMutationService) WithdrawReset(_ context.Context, _ auth.Toko, username *string, _ bool) (*nexusplayers.WithdrawResetResult, error) {
	if username != nil {
		f.lastUsername = *username
	}
	return &nexusplayers.WithdrawResetResult{
		AgentBalance: 100000,
		User: &nexusplayers.WithdrawResetUser{
			Username:       "shareduser",
			WithdrawAmount: 25000,
			Balance:        50000,
		},
	}, f.err
}

func (f *fakePlayerMutationService) TransferStatus(_ context.Context, _ auth.Toko, username string, _ string) (*nexusplayers.TransferStatusResult, error) {
	f.lastUsername = username
	return &nexusplayers.TransferStatusResult{
		Amount:       50000,
		Type:         "user_deposit",
		AgentBalance: 300000,
		Username:     "shareduser",
		UserBalance:  125000,
	}, f.err
}

func TestUserCreateNormalizesUsername(t *testing.T) {
	t.Parallel()

	service := &fakePlayerMutationService{
		createResult: &nexusplayers.CreateUserResult{Username: "shareduser"},
	}
	handler := NewLegacyPlayerMutationAPIHandler(service)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/user/create", strings.NewReader(`{"username":"SharedUser"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.UserCreate(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if service.lastUsername != "shareduser" {
		t.Fatalf("expected normalized username, got %q", service.lastUsername)
	}
}

func TestUserCreateSanitizesUpstreamErrors(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerMutationAPIHandler(&fakePlayerMutationService{
		err: nexusplayers.ErrCreateUserUpstreamFailure,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/user/create", strings.NewReader(`{"username":"shareduser"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.UserCreate(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}

	if strings.Contains(recorder.Body.String(), "secret") {
		t.Fatalf("unexpected leakage: %s", recorder.Body.String())
	}
}

func TestUserDepositReturnsLegacyShape(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerMutationAPIHandler(&fakePlayerMutationService{
		mutationResult: &nexusplayers.MutationResult{
			Username:     "shareduser",
			AgentBalance: 150000,
			UserBalance:  75000,
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/user/deposit", strings.NewReader(`{"username":"SHAREDUSER","amount":50000}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.UserDeposit(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	agent := response["agent"].(map[string]any)
	user := response["user"].(map[string]any)
	if agent["code"] != "Toko A" || agent["balance"] != float64(150000) {
		t.Fatalf("unexpected agent payload: %#v", agent)
	}
	if user["username"] != "shareduser" || user["balance"] != float64(75000) {
		t.Fatalf("unexpected user payload: %#v", user)
	}
}

func TestUserDepositMapsInsufficientBalanceTo400(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerMutationAPIHandler(&fakePlayerMutationService{
		err: nexusplayers.ErrInsufficientNexusBalance,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/user/deposit", strings.NewReader(`{"username":"shareduser","amount":50000}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.UserDeposit(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `Insufficient balance`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestUserWithdrawMapsPlayerNotFoundTo404(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerMutationAPIHandler(&fakePlayerMutationService{
		err: players.ErrNotFound,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/user/withdraw", strings.NewReader(`{"username":"shareduser","amount":50000}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.UserWithdraw(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestUserWithdrawMapsUpstreamBalanceFailureTo400(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerMutationAPIHandler(&fakePlayerMutationService{
		err: nexusplayers.ErrUpstreamUserInsufficientFunds,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/user/withdraw", strings.NewReader(`{"username":"shareduser","amount":50000}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.UserWithdraw(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `User has insufficient balance on upstream platform`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestUserWithdrawSanitizesUnexpectedErrors(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerMutationAPIHandler(&fakePlayerMutationService{
		err: errors.New("boom"),
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/user/withdraw", strings.NewReader(`{"username":"shareduser","amount":50000}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.UserWithdraw(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}

func TestUserWithdrawResetMapsLegacyShape(t *testing.T) {
	t.Parallel()

	handler := NewLegacyPlayerMutationAPIHandler(&fakePlayerMutationService{})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/user/withdraw-reset", strings.NewReader(`{"username":"SHAREDUSER"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.UserWithdrawReset(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `"withdraw_amount":25000`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestTransferStatusMapsLegacyShape(t *testing.T) {
	t.Parallel()

	service := &fakePlayerMutationService{}
	handler := NewLegacyPlayerMutationAPIHandler(service)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/transfer/status", strings.NewReader(`{"username":"SHAREDUSER","agent_sign":"SIG-001"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.TransferStatus(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if service.lastUsername != "shareduser" {
		t.Fatalf("expected normalized username, got %q", service.lastUsername)
	}

	if !strings.Contains(recorder.Body.String(), `"type":"user_deposit"`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
