package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/jobs"
)

type fakeWebhookQueue struct {
	qrisPayload         *jobs.QRISCallbackPayload
	disbursementPayload *jobs.DisbursementCallbackPayload
	err                 error
}

func (f *fakeWebhookQueue) EnqueueProcessQRISCallback(_ context.Context, payload jobs.QRISCallbackPayload) error {
	f.qrisPayload = &payload
	return f.err
}

func (f *fakeWebhookQueue) EnqueueProcessDisbursementCallback(_ context.Context, payload jobs.DisbursementCallbackPayload) error {
	f.disbursementPayload = &payload
	return f.err
}

func TestWebhookHandlerQRISQueuesTask(t *testing.T) {
	queue := &fakeWebhookQueue{}
	handler := NewWebhookHandler(config.Config{
		Integrations: config.IntegrationsConfig{
			QRIS: config.QRISConfig{GlobalUUID: "merchant-uuid"},
		},
	}, queue)

	req := httptest.NewRequest(http.MethodPost, "/api/webhook/qris", bytes.NewBufferString(`{
		"amount": 125000,
		"terminal_id": " player-01 ",
		"merchant_id": "merchant-uuid",
		"trx_id": " trx-001 ",
		"custom_ref": " ref-001 ",
		"status": "SUCCESS"
	}`))
	recorder := httptest.NewRecorder()

	handler.QRIS(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response["status"] != true {
		t.Fatalf("expected status=true, got %#v", response["status"])
	}
	if response["message"] != "OK" {
		t.Fatalf("expected message OK, got %#v", response["message"])
	}

	if queue.qrisPayload == nil {
		t.Fatal("expected qris payload to be queued")
	}
	if queue.qrisPayload.TerminalID != "player-01" {
		t.Fatalf("expected trimmed terminal_id, got %q", queue.qrisPayload.TerminalID)
	}
	if queue.qrisPayload.TrxID != "trx-001" {
		t.Fatalf("expected trimmed trx_id, got %q", queue.qrisPayload.TrxID)
	}
	if queue.qrisPayload.Status != "success" {
		t.Fatalf("expected lowered status, got %q", queue.qrisPayload.Status)
	}
	if queue.qrisPayload.CustomRef == nil || *queue.qrisPayload.CustomRef != "ref-001" {
		t.Fatalf("expected custom_ref to be trimmed, got %#v", queue.qrisPayload.CustomRef)
	}
}

func TestWebhookHandlerQRISValidationError(t *testing.T) {
	queue := &fakeWebhookQueue{}
	handler := NewWebhookHandler(config.Config{
		Integrations: config.IntegrationsConfig{
			QRIS: config.QRISConfig{GlobalUUID: "merchant-uuid"},
		},
	}, queue)

	req := httptest.NewRequest(http.MethodPost, "/api/webhook/qris", bytes.NewBufferString(`{
		"amount": 0,
		"terminal_id": "",
		"merchant_id": "wrong",
		"trx_id": "",
		"status": "unknown"
	}`))
	recorder := httptest.NewRecorder()

	handler.QRIS(recorder, req)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", recorder.Code)
	}
	if queue.qrisPayload != nil {
		t.Fatal("expected qris payload not to be queued on validation failure")
	}
}

func TestWebhookHandlerDisbursementQueuesTask(t *testing.T) {
	queue := &fakeWebhookQueue{}
	handler := NewWebhookHandler(config.Config{
		Integrations: config.IntegrationsConfig{
			QRIS: config.QRISConfig{GlobalUUID: "merchant-uuid"},
		},
	}, queue)

	req := httptest.NewRequest(http.MethodPost, "/api/webhook/disbursement", bytes.NewBufferString(`{
		"amount": 50000,
		"partner_ref_no": " wd-001 ",
		"merchant_id": "merchant-uuid",
		"status": "FAILED",
		"transaction_date": "2026-04-17T12:00:00+07:00"
	}`))
	recorder := httptest.NewRecorder()

	handler.Disbursement(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if queue.disbursementPayload == nil {
		t.Fatal("expected disbursement payload to be queued")
	}
	if queue.disbursementPayload.PartnerRefNo != "wd-001" {
		t.Fatalf("expected trimmed partner_ref_no, got %q", queue.disbursementPayload.PartnerRefNo)
	}
	if queue.disbursementPayload.Status != "failed" {
		t.Fatalf("expected lowered status, got %q", queue.disbursementPayload.Status)
	}
}
