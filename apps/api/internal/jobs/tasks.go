package jobs

const (
	TaskProcessQRISCallback         = "webhook.process_qris_callback"
	TaskProcessDisbursementCallback = "webhook.process_disbursement_callback"
	TaskExpirePendingTransaction    = "transactions.expire_pending"
	TaskRelayTokoCallback           = "callback.relay_toko"
)

type QRISCallbackPayload struct {
	Amount     int64   `json:"amount"`
	TerminalID string  `json:"terminal_id"`
	MerchantID string  `json:"merchant_id"`
	TrxID      string  `json:"trx_id"`
	RRN        *string `json:"rrn,omitempty"`
	CustomRef  *string `json:"custom_ref,omitempty"`
	Vendor     *string `json:"vendor,omitempty"`
	Status     string  `json:"status"`
	CreatedAt  *string `json:"created_at,omitempty"`
	FinishAt   *string `json:"finish_at,omitempty"`
}

type DisbursementCallbackPayload struct {
	Amount          int64   `json:"amount"`
	PartnerRefNo    string  `json:"partner_ref_no"`
	Status          string  `json:"status"`
	TransactionDate *string `json:"transaction_date,omitempty"`
	MerchantID      string  `json:"merchant_id"`
}

type ExpirePendingTransactionPayload struct {
	TransactionID int64 `json:"transaction_id"`
}

type TokoCallbackPayload struct {
	CallbackURL *string        `json:"callback_url,omitempty"`
	Payload     map[string]any `json:"payload"`
	EventType   string         `json:"event_type"`
	Reference   string         `json:"reference"`
}
