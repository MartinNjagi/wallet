package data

import "time"

type WalletBalanceResponse struct {
	ClientID   int64  `json:"client_id"`
	PaymentRef string `json:"payment_ref"`
	Balance    int64  `json:"balance"`
	Currency   string `json:"currency"`
}

type WalletTransactionResponse struct {
	ID              uint      `json:"id"`
	ClientID        uint      `json:"client_id"`
	Amount          int64     `json:"amount"`
	TransactionType string    `json:"transaction_type"`
	ReferenceType   string    `json:"reference_type"`
	ReferenceID     string    `json:"reference_id"`
	Description     string    `json:"description"`
	FiatAmount      float64   `json:"fiat_amount"`
	CreatedAt       time.Time `json:"created_at"`
}

type MpesaTransactionResponse struct {
	ID                uint      `json:"id"`
	ClientID          uint      `json:"client_id"`
	CheckoutRequestID string    `json:"checkout_request_id"`
	Amount            float64   `json:"amount"`
	Credits           int64     `json:"credits"`
	Status            string    `json:"status"`
	ReceiptNumber     string    `json:"receipt_number"`
	CreatedAt         time.Time `json:"created_at"`
}

type AdminWalletSummaryResponse struct {
	TotalSystemBalance  int64   `json:"total_system_balance"`  // Sum of all active wallets
	TotalFiatProcessed  float64 `json:"total_fiat_processed"`  // Sum of all SUCCESS M-pesa tx
	TotalCreditsBurned  int64   `json:"total_credits_burned"`  // Absolute sum of all DEBIT tx
	TotalCreditsGranted int64   `json:"total_credits_granted"` // Sum of all CREDIT tx (TopUps, Refunds, Adjs)
}
