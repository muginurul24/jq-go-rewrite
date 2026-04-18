package qris

type GenerateResponse struct {
	Status    bool `json:"status"`
	Data      any  `json:"data,omitempty"`
	TrxID     any  `json:"trx_id,omitempty"`
	ExpiredAt any  `json:"expired_at,omitempty"`
	Error     any  `json:"error,omitempty"`
}

type CheckStatusResponse struct {
	Status     any `json:"status,omitempty"`
	Amount     any `json:"amount,omitempty"`
	MerchantID any `json:"merchant_id,omitempty"`
	TrxID      any `json:"trx_id,omitempty"`
	RRN        any `json:"rrn,omitempty"`
	CreatedAt  any `json:"created_at,omitempty"`
	FinishAt   any `json:"finish_at,omitempty"`
	Error      any `json:"error,omitempty"`
}

type BalanceResponse struct {
	Status         string `json:"status"`
	PendingBalance any    `json:"pending_balance,omitempty"`
	SettleBalance  any    `json:"settle_balance,omitempty"`
	Error          any    `json:"error,omitempty"`
}

type InquiryResponse struct {
	Status bool         `json:"status"`
	Data   *InquiryData `json:"data,omitempty"`
	Error  any          `json:"error,omitempty"`
}

type InquiryData struct {
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	BankCode      string `json:"bank_code"`
	BankName      string `json:"bank_name"`
	PartnerRefNo  string `json:"partner_ref_no"`
	VendorRefNo   string `json:"vendor_ref_no"`
	Amount        int64  `json:"amount"`
	Fee           int64  `json:"fee"`
	InquiryID     int64  `json:"inquiry_id"`
}

type TransferResponse struct {
	Status bool `json:"status"`
	Error  any  `json:"error,omitempty"`
}
