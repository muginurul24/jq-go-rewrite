package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/transactions"
)

type fakeBackofficeTransactionsService struct {
	listResult   *transactions.AdminListResult
	detailResult *transactions.AdminTransactionDetail
	exportRows   []transactions.AdminTransactionRecord
	err          error
}

func (f *fakeBackofficeTransactionsService) ListForBackoffice(_ context.Context, _ auth.PublicUser, _ transactions.AdminListInput) (*transactions.AdminListResult, error) {
	return f.listResult, f.err
}

func (f *fakeBackofficeTransactionsService) FindDetailForBackoffice(_ context.Context, _ auth.PublicUser, _ int64) (*transactions.AdminTransactionDetail, error) {
	return f.detailResult, f.err
}

func (f *fakeBackofficeTransactionsService) ExportForBackoffice(_ context.Context, _ auth.PublicUser, _ transactions.AdminListInput) ([]transactions.AdminTransactionRecord, error) {
	return f.exportRows, f.err
}

func TestBackofficeTransactionsListReturnsFrontendCompatibleFilterOptions(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeTransactionsHandler(&fakeBackofficeTransactionsService{
		listResult: &transactions.AdminListResult{
			Items: []transactions.AdminTransactionRecord{
				{
					ID:            11,
					TokoID:        7,
					TokoName:      "Toko Alpha",
					OwnerUsername: "owner-a",
					Category:      "qris",
					Type:          "deposit",
					Status:        "success",
					Amount:        125000,
					CreatedAt:     time.Unix(1_700_000_000, 0),
					UpdatedAt:     time.Unix(1_700_000_100, 0),
				},
			},
			Page:       1,
			PerPage:    25,
			Total:      1,
			TotalPages: 1,
			TokoOptions: []transactions.AdminTokoOption{
				{
					ID:            7,
					Name:          "Toko Alpha",
					OwnerUsername: "owner-a",
				},
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/transactions", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID:   1,
		Role: "dev",
	}))
	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	filters := payload["filters"].(map[string]any)
	tokos := filters["tokos"].([]any)
	if len(tokos) != 1 {
		t.Fatalf("expected one toko option, got %#v", filters["tokos"])
	}

	first := tokos[0].(map[string]any)
	if first["name"] != "Toko Alpha" || first["ownerUsername"] != "owner-a" {
		t.Fatalf("expected frontend-compatible transaction filter, got %#v", first)
	}
}

func TestBackofficeTransactionsExportCSV(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeTransactionsHandler(&fakeBackofficeTransactionsService{
		exportRows: []transactions.AdminTransactionRecord{
			{
				ID:             11,
				TokoID:         7,
				TokoName:       "Toko Alpha",
				OwnerUsername:  "owner-a",
				Player:         stringPtr("player-a"),
				ExternalPlayer: stringPtr("ext-a"),
				Category:       "qris",
				Type:           "deposit",
				Status:         "success",
				Amount:         125000,
				Code:           stringPtr("REF-001"),
				Note:           stringPtr(`{"purpose":"generate"}`),
				CreatedAt:      time.Unix(1_700_000_000, 0),
				UpdatedAt:      time.Unix(1_700_000_100, 0),
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/transactions/export?format=csv", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID:   1,
		Role: "dev",
	}))
	recorder := httptest.NewRecorder()

	handler.Export(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/csv") {
		t.Fatalf("expected csv content type, got %q", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Toko Alpha") || !strings.Contains(body, "REF-001") {
		t.Fatalf("expected csv body to contain export row, got %q", body)
	}
}

func TestBackofficeTransactionsExportXLSX(t *testing.T) {
	t.Parallel()

	handler := NewBackofficeTransactionsHandler(&fakeBackofficeTransactionsService{
		exportRows: []transactions.AdminTransactionRecord{
			{
				ID:            12,
				TokoID:        9,
				TokoName:      "Toko Bravo",
				OwnerUsername: "owner-b",
				Category:      "nexusggr",
				Type:          "withdrawal",
				Status:        "pending",
				Amount:        99000,
				CreatedAt:     time.Unix(1_700_100_000, 0),
				UpdatedAt:     time.Unix(1_700_100_100, 0),
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/transactions/export?format=xlsx", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID:   1,
		Role: "superadmin",
	}))
	recorder := httptest.NewRecorder()

	handler.Export(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "spreadsheetml") {
		t.Fatalf("expected xlsx content type, got %q", got)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("expected xlsx payload")
	}
}

func stringPtr(value string) *string {
	return &value
}
