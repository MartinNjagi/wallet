package data

import "time"

// --- Responses ---

type WalletBalanceResponse struct {
	ClientID int64  `json:"client_id"`
	Balance  int64  `json:"balance"`
	Currency string `json:"currency"`
}

type WalletTransactionResponse struct {
	ID              uint      `json:"id"`
	Amount          int64     `json:"amount"`
	TransactionType string    `json:"transaction_type"`
	ReferenceType   string    `json:"reference_type"`
	ReferenceID     string    `json:"reference_id"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
}

type APIResponse struct {
	Status     int             `json:"status"`
	Message    string          `json:"message,omitempty"`
	Data       interface{}     `json:"data,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}
