package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/withdrawals"
)

type fakeBackofficeWithdrawalService struct {
	bootstrapResult *withdrawals.BootstrapResult
	inquiryResult   *withdrawals.InquiryResult
	submitResult    *withdrawals.SubmitResult
	err             error
}

func (f *fakeBackofficeWithdrawalService) Bootstrap(_ context.Context, _ auth.PublicUser, _ *int64) (*withdrawals.BootstrapResult, error) {
	return f.bootstrapResult, f.err
}

func (f *fakeBackofficeWithdrawalService) Inquiry(_ context.Context, _ auth.PublicUser, _ int64, _ int64, _ int64) (*withdrawals.InquiryResult, error) {
	return f.inquiryResult, f.err
}

func (f *fakeBackofficeWithdrawalService) Submit(_ context.Context, _ auth.PublicUser, _ int64, _ int64, _ int64, _ int64) (*withdrawals.SubmitResult, error) {
	return f.submitResult, f.err
}

func TestBackofficeWithdrawalBootstrapReturnsShape(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeWithdrawalHandler(&fakeBackofficeWithdrawalService{
		bootstrapResult: &withdrawals.BootstrapResult{
			Tokos: []withdrawals.TokoOption{
				{ID: 7, Name: "Toko Alpha", OwnerUsername: "owner-a", SettleBalance: 250000},
			},
			SelectedToko:  &withdrawals.TokoOption{ID: 7, Name: "Toko Alpha", OwnerUsername: "owner-a", SettleBalance: 250000},
			Banks:         []withdrawals.BankOption{{ID: 8, BankCode: "014", BankName: "BCA", AccountNumber: "1234567890", AccountName: "Owner A"}},
			FeePercentage: 15,
			MinimumAmount: 25000,
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/withdrawal/bootstrap?toko_id=7", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 1, Role: "dev"}))
	recorder := httptest.NewRecorder()

	handler.Bootstrap(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `"feePercentage":15`) || !strings.Contains(body, `"ownerUsername":"owner-a"`) {
		t.Fatalf("unexpected bootstrap response: %s", body)
	}
}

func TestBackofficeWithdrawalInquiryValidatesPayload(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeWithdrawalHandler(&fakeBackofficeWithdrawalService{})
	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/withdrawal/inquiry", strings.NewReader(`{"tokoId":0,"bankId":0,"amount":0}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 10, Role: "admin"}))
	recorder := httptest.NewRecorder()

	handler.Inquiry(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `"tokoId":"Toko wajib dipilih."`) || !strings.Contains(body, `"bankId":"Rekening tujuan wajib dipilih."`) {
		t.Fatalf("unexpected validation response: %s", body)
	}
}

func TestBackofficeWithdrawalSubmitMapsInquiryStateNotFound(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeWithdrawalHandler(&fakeBackofficeWithdrawalService{
		err: withdrawals.ErrInquiryStateNotFound,
	})
	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/withdrawal/submit", strings.NewReader(`{"tokoId":7,"bankId":8,"amount":50000,"inquiryId":123}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 10, Role: "admin"}))
	recorder := httptest.NewRecorder()

	handler.Submit(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), "Data inquiry tidak ditemukan") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
