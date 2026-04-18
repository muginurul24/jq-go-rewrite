package queue

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/jobs"
)

type fakeQRISProcessor struct {
	payload *jobs.QRISCallbackPayload
}

func (f *fakeQRISProcessor) ProcessQRISCallback(_ context.Context, payload jobs.QRISCallbackPayload) error {
	f.payload = &payload
	return nil
}

type fakeDisbursementProcessor struct {
	payload *jobs.DisbursementCallbackPayload
}

func (f *fakeDisbursementProcessor) ProcessDisbursementCallback(_ context.Context, payload jobs.DisbursementCallbackPayload) error {
	f.payload = &payload
	return nil
}

type fakePendingExpirer struct {
	transactionID int64
}

func (f *fakePendingExpirer) ExpirePendingTransaction(_ context.Context, transactionID int64) error {
	f.transactionID = transactionID
	return nil
}

type fakeRelayer struct {
	payload *jobs.TokoCallbackPayload
}

func (f *fakeRelayer) RelayTokoCallback(_ context.Context, payload jobs.TokoCallbackPayload) error {
	f.payload = &payload
	return nil
}

func TestWorkerHandlerProcessQRISCallback(t *testing.T) {
	processor := &fakeQRISProcessor{}
	handler := &Handler{
		logger:        zerolog.Nop(),
		qrisProcessor: processor,
	}

	task := asynq.NewTask(jobs.TaskProcessQRISCallback, mustJSON(t, jobs.QRISCallbackPayload{
		Amount:     1000,
		TerminalID: "player-01",
		MerchantID: "merchant-uuid",
		TrxID:      "trx-001",
		Status:     "success",
	}))

	if err := handler.HandleProcessQRISCallback(context.Background(), task); err != nil {
		t.Fatalf("handle qris callback: %v", err)
	}
	if processor.payload == nil || processor.payload.TrxID != "trx-001" {
		t.Fatalf("expected qris payload to be forwarded, got %#v", processor.payload)
	}
}

func TestWorkerHandlerProcessDisbursementCallback(t *testing.T) {
	processor := &fakeDisbursementProcessor{}
	handler := &Handler{
		logger:                zerolog.Nop(),
		disbursementProcessor: processor,
	}

	task := asynq.NewTask(jobs.TaskProcessDisbursementCallback, mustJSON(t, jobs.DisbursementCallbackPayload{
		Amount:       2000,
		PartnerRefNo: "wd-001",
		Status:       "failed",
		MerchantID:   "merchant-uuid",
	}))

	if err := handler.HandleProcessDisbursementCallback(context.Background(), task); err != nil {
		t.Fatalf("handle disbursement callback: %v", err)
	}
	if processor.payload == nil || processor.payload.PartnerRefNo != "wd-001" {
		t.Fatalf("expected disbursement payload to be forwarded, got %#v", processor.payload)
	}
}

func TestWorkerHandlerExpirePendingTransaction(t *testing.T) {
	expirer := &fakePendingExpirer{}
	handler := &Handler{
		logger:         zerolog.Nop(),
		pendingExpirer: expirer,
	}

	task := asynq.NewTask(jobs.TaskExpirePendingTransaction, mustJSON(t, jobs.ExpirePendingTransactionPayload{
		TransactionID: 99,
	}))

	if err := handler.HandleExpirePendingTransaction(context.Background(), task); err != nil {
		t.Fatalf("handle expire pending transaction: %v", err)
	}
	if expirer.transactionID != 99 {
		t.Fatalf("expected transaction id 99, got %d", expirer.transactionID)
	}
}

func TestWorkerHandlerRelayTokoCallback(t *testing.T) {
	relayer := &fakeRelayer{}
	handler := &Handler{
		logger:              zerolog.Nop(),
		tokoCallbackRelayer: relayer,
	}

	task := asynq.NewTask(jobs.TaskRelayTokoCallback, mustJSON(t, jobs.TokoCallbackPayload{
		CallbackURL: stringPointer("https://example.com/callback"),
		EventType:   "qris",
		Reference:   "trx-001",
		Payload: map[string]any{
			"trx_id": "trx-001",
			"status": "success",
		},
	}))

	if err := handler.HandleRelayTokoCallback(context.Background(), task); err != nil {
		t.Fatalf("handle relay toko callback: %v", err)
	}
	if relayer.payload == nil || relayer.payload.Reference != "trx-001" {
		t.Fatalf("expected relay payload to be forwarded, got %#v", relayer.payload)
	}
}

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()

	taskPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return taskPayload
}

func stringPointer(value string) *string {
	return &value
}
