package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/jobs"
)

func TestCalculateRegularQRISAmounts(t *testing.T) {
	pendingDelta, incomeDelta := calculateRegularQRISAmounts(100000, 3)

	if pendingDelta != 97000 {
		t.Fatalf("expected pending delta 97000, got %d", pendingDelta)
	}
	if incomeDelta != 3000 {
		t.Fatalf("expected income delta 3000, got %d", incomeDelta)
	}
}

func TestCalculateNexusTopupAmounts(t *testing.T) {
	nexusDelta, incomeDelta := calculateNexusTopupAmounts(100000)

	if nexusDelta != 1428571 {
		t.Fatalf("expected nexus delta 1428571, got %d", nexusDelta)
	}
	if incomeDelta != 98200 {
		t.Fatalf("expected income delta 98200, got %d", incomeDelta)
	}

	nexusDelta, incomeDelta = calculateNexusTopupAmounts(1_500_000)
	if nexusDelta != 25000000 {
		t.Fatalf("expected discounted nexus delta 25000000, got %d", nexusDelta)
	}
	if incomeDelta != 1498200 {
		t.Fatalf("expected discounted income delta 1498200, got %d", incomeDelta)
	}
}

func TestBuildQRISNoteRegularDeposit(t *testing.T) {
	customRef := "ref-001"
	rrn := "rrn-001"
	vendor := "qris"
	finishAt := "2026-04-17T12:00:00+07:00"

	note := buildQRISNote(map[string]any{
		"purpose":    "generate",
		"expired_at": 1713330000,
	}, jobs.QRISCallbackPayload{
		CustomRef: &customRef,
		RRN:       &rrn,
		Vendor:    &vendor,
		FinishAt:  &finishAt,
	}, false)

	if _, ok := note["expired_at"]; ok {
		t.Fatal("expected regular deposit note not to preserve expired_at")
	}
	if note["custom_ref"] != &customRef {
		t.Fatalf("expected custom_ref pointer to be preserved, got %#v", note["custom_ref"])
	}
}

func TestBuildQRISNoteNexusTopupPreservesExistingFields(t *testing.T) {
	rrn := "rrn-001"
	vendor := "qris"
	finishAt := "2026-04-17T12:00:00+07:00"

	note := buildQRISNote(map[string]any{
		"purpose":    "nexusggr_topup",
		"expired_at": 1713330000,
		"qris_data":  "000201...",
	}, jobs.QRISCallbackPayload{
		RRN:      &rrn,
		Vendor:   &vendor,
		FinishAt: &finishAt,
	}, true)

	if note["qris_data"] != "000201..." {
		t.Fatalf("expected qris_data to be preserved, got %#v", note["qris_data"])
	}
	if note["rrn"] != &rrn {
		t.Fatalf("expected rrn pointer to be preserved, got %#v", note["rrn"])
	}
}

func TestRelayTokoCallbackPostsJSONPayload(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode callback payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewService(nil, zerolog.Nop(), nil)
	callbackURL := server.URL

	err := service.RelayTokoCallback(context.Background(), jobs.TokoCallbackPayload{
		CallbackURL: &callbackURL,
		EventType:   "qris",
		Reference:   "trx-001",
		Payload: map[string]any{
			"trx_id": "trx-001",
			"status": "success",
		},
	})
	if err != nil {
		t.Fatalf("relay toko callback: %v", err)
	}
	if received["trx_id"] != "trx-001" {
		t.Fatalf("expected trx_id to be forwarded, got %#v", received["trx_id"])
	}
}

func TestNormalizeQRISCallbackStatus(t *testing.T) {
	testCases := map[string]string{
		"pending":  "pending",
		"success":  "success",
		"paid":     "success",
		" failed ": "failed",
		"expired":  "expired",
		"noop":     "",
	}

	for input, expected := range testCases {
		if actual := normalizeQRISCallbackStatus(input); actual != expected {
			t.Fatalf("normalizeQRISCallbackStatus(%q) = %q, expected %q", input, actual, expected)
		}
	}
}

func TestScheduleOrRelayTokoCallbackFallsBackToDirectRelay(t *testing.T) {
	delivered := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delivered = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewService(nil, zerolog.Nop(), fakeRelayScheduler{err: errors.New("redis down")})
	callbackURL := server.URL

	if err := service.scheduleOrRelayTokoCallback(context.Background(), jobs.TokoCallbackPayload{
		CallbackURL: &callbackURL,
		EventType:   "qris",
		Reference:   "trx-001",
		Payload:     map[string]any{"trx_id": "trx-001", "status": "success"},
	}); err != nil {
		t.Fatalf("scheduleOrRelayTokoCallback should fall back to direct relay: %v", err)
	}

	if !delivered {
		t.Fatal("expected direct relay fallback to deliver callback")
	}
}

type fakeRelayScheduler struct {
	err error
}

func (f fakeRelayScheduler) EnqueueRelayTokoCallback(_ context.Context, _ jobs.TokoCallbackPayload) error {
	return f.err
}
