package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	qrismodule "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/qris"
)

type fakeQrisService struct {
	generateResult *qrismodule.GenerateResult
	checkResult    *qrismodule.CheckStatusResult
	err            error
}

func (f *fakeQrisService) Generate(_ context.Context, _ auth.Toko, _ qrismodule.GenerateParams) (*qrismodule.GenerateResult, error) {
	return f.generateResult, f.err
}

func (f *fakeQrisService) CheckStatus(_ context.Context, _ auth.Toko, _ string) (*qrismodule.CheckStatusResult, error) {
	return f.checkResult, f.err
}

func TestGenerateQrisReturnsLegacyShape(t *testing.T) {
	t.Parallel()

	handler := NewLegacyQrisAPIHandler(&fakeQrisService{
		generateResult: &qrismodule.GenerateResult{
			Data:  "000201010212...",
			TrxID: "TRX-001",
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/generate", strings.NewReader(`{"username":"demo-player","amount":100000,"expire":900,"custom_ref":"REF-001"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.Generate(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `"trx_id":"TRX-001"`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestGenerateQrisSanitizesUpstreamFailure(t *testing.T) {
	t.Parallel()

	handler := NewLegacyQrisAPIHandler(&fakeQrisService{err: qrismodule.ErrGenerateUpstreamFailure})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/generate", strings.NewReader(`{"username":"demo-player","amount":100000}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.Generate(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}

func TestCheckStatusMapsNotFoundTo404(t *testing.T) {
	t.Parallel()

	handler := NewLegacyQrisAPIHandler(&fakeQrisService{err: qrismodule.ErrTransactionNotFound})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/check-status", strings.NewReader(`{"trx_id":"TRX-OTHER"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.CheckStatus(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestCheckStatusReturnsNormalizedStatus(t *testing.T) {
	t.Parallel()

	handler := NewLegacyQrisAPIHandler(&fakeQrisService{
		checkResult: &qrismodule.CheckStatusResult{
			TrxID:  "TRX-001",
			Status: "success",
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/check-status", strings.NewReader(`{"trx_id":"TRX-001"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.CheckStatus(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `"status":"success"`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestCheckStatusSanitizesUnexpectedErrors(t *testing.T) {
	t.Parallel()

	handler := NewLegacyQrisAPIHandler(&fakeQrisService{err: errors.New("boom")})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/check-status", strings.NewReader(`{"trx_id":"TRX-001"}`))
	request = request.WithContext(auth.WithCurrentToko(request.Context(), auth.Toko{ID: 1, Name: "Toko A"}))
	recorder := httptest.NewRecorder()

	handler.CheckStatus(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}
