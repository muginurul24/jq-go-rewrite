package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/callmanagement"
)

type fakeBackofficeCallManagementService struct {
	bootstrapResult *callmanagement.BootstrapResult
	activePlayers   []callmanagement.ActivePlayerRecord
	callList        []callmanagement.CallOption
	history         []callmanagement.CallHistoryRecord
	applyResult     any
	cancelResult    any
	controlResult   any
	err             error
}

func (f *fakeBackofficeCallManagementService) Bootstrap(_ context.Context, _ auth.PublicUser) (*callmanagement.BootstrapResult, error) {
	return f.bootstrapResult, f.err
}

func (f *fakeBackofficeCallManagementService) ActivePlayers(_ context.Context, _ auth.PublicUser) ([]callmanagement.ActivePlayerRecord, error) {
	return f.activePlayers, f.err
}

func (f *fakeBackofficeCallManagementService) CallList(_ context.Context, _ string, _ string) ([]callmanagement.CallOption, error) {
	return f.callList, f.err
}

func (f *fakeBackofficeCallManagementService) Apply(_ context.Context, _ auth.PublicUser, _ callmanagement.ApplyInput) (any, error) {
	return f.applyResult, f.err
}

func (f *fakeBackofficeCallManagementService) History(_ context.Context, _ auth.PublicUser, _ int, _ int) ([]callmanagement.CallHistoryRecord, error) {
	return f.history, f.err
}

func (f *fakeBackofficeCallManagementService) Cancel(_ context.Context, _ int) (any, error) {
	return f.cancelResult, f.err
}

func (f *fakeBackofficeCallManagementService) ControlRTP(_ context.Context, _ auth.PublicUser, _ callmanagement.ControlRTPInput) (any, error) {
	return f.controlResult, f.err
}

func (f *fakeBackofficeCallManagementService) ControlUsersRTP(_ context.Context, _ auth.PublicUser, _ float64) (any, error) {
	return f.controlResult, f.err
}

func TestBackofficeCallManagementActivePlayersReturnsMappedRows(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeCallManagementHandler(&fakeBackofficeCallManagementService{
		activePlayers: []callmanagement.ActivePlayerRecord{
			{
				PlayerID:     7,
				Username:     "shareduser",
				UserLabel:    "shareduser (Toko Alpha)",
				TokoName:     "Toko Alpha",
				ProviderCode: "PG",
				GameCode:     "G1",
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/call-management/active-players", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 1, Role: "dev"}))
	recorder := httptest.NewRecorder()

	handler.ActivePlayers(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"userLabel":"shareduser (Toko Alpha)"`) {
		t.Fatalf("unexpected active players response: %s", recorder.Body.String())
	}
}

func TestBackofficeCallManagementApplyValidatesPayload(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeCallManagementHandler(&fakeBackofficeCallManagementService{})
	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/call-management/apply", strings.NewReader(`{"playerId":0,"providerCode":"","gameCode":"","callTypeValue":0}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 10, Role: "admin"}))
	recorder := httptest.NewRecorder()

	handler.Apply(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"playerId":"Player wajib dipilih."`) {
		t.Fatalf("unexpected validation response: %s", recorder.Body.String())
	}
}

func TestBackofficeCallManagementHistoryMapsUnavailableTo502(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeCallManagementHandler(&fakeBackofficeCallManagementService{
		err: callmanagement.ErrCallHistoryUnavailable,
	})
	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/call-management/history", strings.NewReader(`{"offset":0,"limit":100}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 10, Role: "admin"}))
	recorder := httptest.NewRecorder()

	handler.History(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", recorder.Code)
	}
}

func TestBackofficeCallManagementCancelReturnsCanceledMoney(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeCallManagementHandler(&fakeBackofficeCallManagementService{
		cancelResult: 12500,
	})
	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/call-management/cancel", strings.NewReader(`{"callId":77}`))
	recorder := httptest.NewRecorder()

	handler.Cancel(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"canceledMoney":12500`) {
		t.Fatalf("unexpected cancel response: %s", recorder.Body.String())
	}
}
