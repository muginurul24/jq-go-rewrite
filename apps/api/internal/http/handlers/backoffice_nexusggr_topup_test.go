package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/nexusggrtopup"
)

type fakeBackofficeTopupService struct {
	bootstrapResult *nexusggrtopup.BootstrapResult
	generateResult  *nexusggrtopup.GenerateResult
	statusResult    *nexusggrtopup.StatusResult
	err             error
}

func (f *fakeBackofficeTopupService) Bootstrap(_ context.Context, _ auth.PublicUser, _ *int64) (*nexusggrtopup.BootstrapResult, error) {
	return f.bootstrapResult, f.err
}

func (f *fakeBackofficeTopupService) Generate(_ context.Context, _ auth.PublicUser, _ int64, _ int64) (*nexusggrtopup.GenerateResult, error) {
	return f.generateResult, f.err
}

func (f *fakeBackofficeTopupService) CheckStatus(_ context.Context, _ auth.PublicUser, _ int64, _ string) (*nexusggrtopup.StatusResult, error) {
	return f.statusResult, f.err
}

func TestBackofficeTopupBootstrapUsesFrontendCompatibleJSONKeys(t *testing.T) {
	t.Parallel()

	expiresAt := int64(1_776_000_000)
	handler := NewBackofficeNexusggrTopupHandler(&fakeBackofficeTopupService{
		bootstrapResult: &nexusggrtopup.BootstrapResult{
			Tokos: []nexusggrtopup.TokoOption{
				{
					ID:              7,
					Name:            "Toko Alpha",
					OwnerUsername:   "owner-a",
					NexusggrBalance: 125000,
				},
			},
			SelectedToko: &nexusggrtopup.TokoOption{
				ID:              7,
				Name:            "Toko Alpha",
				OwnerUsername:   "owner-a",
				NexusggrBalance: 125000,
			},
			TopupRatio: 7,
			PendingTopup: &nexusggrtopup.PendingTopup{
				Amount:          100000,
				TransactionCode: "TRX-001",
				ExpiresAt:       &expiresAt,
				Status:          "pending",
				QrPayload:       "000201010212",
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/backoffice/api/nexusggr-topup/bootstrap", nil)
	request = request.WithContext(auth.WithCurrentUser(request.Context(), auth.PublicUser{
		ID: 1, Role: "dev",
	}))
	recorder := httptest.NewRecorder()

	handler.Bootstrap(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		`"name":"Toko Alpha"`,
		`"ownerUsername":"owner-a"`,
		`"nexusggrBalance":125000`,
		`"transactionCode":"TRX-001"`,
		`"qrPayload":"000201010212"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected response to contain %s, got %s", expected, body)
		}
	}
}
