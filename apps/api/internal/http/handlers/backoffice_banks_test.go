package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/banks"
)

type fakeBackofficeBanksService struct {
	listResult    *banks.AdminListResult
	createResult  *banks.AdminBankRecord
	updateResult  *banks.AdminBankRecord
	inquiryResult *banks.InquiryResult
	err           error
}

func (f *fakeBackofficeBanksService) ListForBackoffice(_ context.Context, _ auth.PublicUser, _ banks.AdminListInput) (*banks.AdminListResult, error) {
	return f.listResult, f.err
}

func (f *fakeBackofficeBanksService) CreateForBackoffice(_ context.Context, _ auth.PublicUser, _ banks.CreateInput) (*banks.AdminBankRecord, error) {
	return f.createResult, f.err
}

func (f *fakeBackofficeBanksService) UpdateForBackoffice(_ context.Context, _ auth.PublicUser, _ int64, _ banks.UpdateInput) (*banks.AdminBankRecord, error) {
	return f.updateResult, f.err
}

func (f *fakeBackofficeBanksService) InquiryForBackoffice(_ context.Context, _ auth.PublicUser, _ int64, _ string, _ string) (*banks.InquiryResult, error) {
	return f.inquiryResult, f.err
}

func TestBackofficeBanksListReturnsRowsAndOwners(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeBanksHandler(&fakeBackofficeBanksService{
		listResult: &banks.AdminListResult{
			Data: []banks.AdminBankRecord{
				{
					ID:            8,
					UserID:        10,
					OwnerUsername: "owner-a",
					OwnerName:     "Owner A",
					BankCode:      "014",
					BankName:      "BCA",
					AccountNumber: "1234567890",
					AccountName:   "Owner A",
					CreatedAt:     time.Unix(1_700_000_000, 0),
					UpdatedAt:     time.Unix(1_700_000_100, 0),
				},
			},
			Page:       1,
			PerPage:    25,
			Total:      1,
			TotalPages: 1,
			Owners: []banks.OwnerOption{
				{ID: 10, Username: "owner-a", Name: "Owner A"},
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/banks", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 1, Role: "dev"}))
	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `"bankName":"BCA"`) || !strings.Contains(body, `"ownerUsername":"owner-a"`) {
		t.Fatalf("unexpected list response: %s", body)
	}
}

func TestBackofficeBanksCreateValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeBanksHandler(&fakeBackofficeBanksService{})
	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/banks", strings.NewReader(`{"bankCode":"","accountNumber":"","accountName":""}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 1, Role: "dev"}))
	recorder := httptest.NewRecorder()

	handler.Create(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `"userId":"Owner is required."`) || !strings.Contains(body, `"accountName":"Nama rekening wajib diisi."`) {
		t.Fatalf("unexpected validation response: %s", body)
	}
}

func TestBackofficeBanksUpdateMapsDuplicateAccountNumber(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeBanksHandler(&fakeBackofficeBanksService{
		err: banks.ErrDuplicateAccountNumber,
	})
	request := httptest.NewRequest(http.MethodPatch, "/backoffice/api/banks/8", strings.NewReader(`{"bankCode":"014","accountNumber":"123","accountName":"Owner A"}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 10, Role: "admin"}))
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(request.Context()).URLParams.Add("bankID", "8")
	recorder := httptest.NewRecorder()

	handler.Update(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), "Nomor rekening sudah dipakai") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestBackofficeBanksInquiryReturnsShape(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeBanksHandler(&fakeBackofficeBanksService{
		inquiryResult: &banks.InquiryResult{
			BankCode:      "014",
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountName:   "Owner A",
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/backoffice/api/banks/inquiry", strings.NewReader(`{"bankCode":"014","accountNumber":"1234567890"}`))
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 10, Role: "admin"}))
	recorder := httptest.NewRecorder()

	handler.Inquiry(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `"accountName":"Owner A"`) {
		t.Fatalf("unexpected inquiry response: %s", recorder.Body.String())
	}
}
