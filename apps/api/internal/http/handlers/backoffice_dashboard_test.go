package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/dashboard"
)

type fakeBackofficeDashboardService struct {
	overviewResult *dashboard.OverviewResult
	pulseResult    *dashboard.OperationalPulseResult
	err            error
}

func (f *fakeBackofficeDashboardService) Overview(_ context.Context, _ auth.PublicUser) (*dashboard.OverviewResult, error) {
	return f.overviewResult, f.err
}

func (f *fakeBackofficeDashboardService) OperationalPulse(_ context.Context, _ auth.PublicUser) (*dashboard.OperationalPulseResult, error) {
	return f.pulseResult, f.err
}

func TestBackofficeDashboardOverviewReturnsShape(t *testing.T) {
	handler := NewBackofficeDashboardHandler(&fakeBackofficeDashboardService{
		overviewResult: &dashboard.OverviewResult{
			GeneratedAt: "2026-04-17T11:00:00Z",
			Role:        "dev",
			Stats: dashboard.OverviewStats{
				PendingBalance: 125000,
			},
			RecentTransactions: []dashboard.RecentTransaction{
				{ID: 1, TokoName: "Toko Alpha", Category: "qris", Type: "deposit", Status: "success", Amount: 100000, CreatedAt: "2026-04-17T10:00:00Z"},
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/dashboard/overview", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 1, Role: "dev"}))
	recorder := httptest.NewRecorder()

	handler.Overview(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{"generatedAt", "pendingBalance", "recentTransactions", "Toko Alpha"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected response body to contain %q, got %s", expected, body)
		}
	}
}

func TestBackofficeDashboardOperationalPulseReturnsShape(t *testing.T) {
	handler := NewBackofficeDashboardHandler(&fakeBackofficeDashboardService{
		pulseResult: &dashboard.OperationalPulseResult{
			GeneratedAt: "2026-04-17T11:00:00Z",
			Role:        "admin",
			Stats: dashboard.OperationalPulseStats{
				PendingTransactions: 3,
			},
			QRIS: []dashboard.TransactionSeriesRow{
				{Date: "2026-04-17", Deposit: 100000, Withdrawal: 50000},
			},
			Nexusggr: []dashboard.TransactionSeriesRow{
				{Date: "2026-04-17", Deposit: 70000, Withdrawal: 30000},
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/dashboard/operational-pulse", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{ID: 2, Role: "admin"}))
	recorder := httptest.NewRecorder()

	handler.OperationalPulse(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{"pendingTransactions", "\"qris\"", "\"nexusggr\""} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(expected)) {
			t.Fatalf("expected response body to contain %q, got %s", expected, body)
		}
	}
}
