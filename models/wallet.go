package models

import (
	"gorm.io/gorm"
)

func (*Wallet) TableName() string {
	return "wallets"
}

func (*WalletTransaction) TableName() string {
	return "wallet_transactions"
}

// Wallet stores the current active balance of a client.
type Wallet struct {
	gorm.Model
	ClientID uint   `gorm:"uniqueIndex;not null"`
	Balance  int64  `gorm:"default:0"` // Storing credits as integers to avoid floating point errors
	Currency string `gorm:"default:'KES'"`
}

// WalletTransaction is the Immutable Ledger tracking every movement of credits.
type WalletTransaction struct {
	gorm.Model
	WalletID        uint    `gorm:"index"`
	ClientID        uint    `gorm:"index"`
	Amount          int64   // Positive for Credit, Negative for Debit
	TransactionType string  // "CREDIT" or "DEBIT"
	ReferenceType   string  // "MPESA", "SMS", "ADJ", "REV"
	ReferenceID     string  `gorm:"uniqueIndex:idx_ref_type_id"` // Composite unique index with ReferenceType for absolute idempotency
	Description     string  `gorm:"type:text"`
	FiatAmount      float64 // The real-world currency value associated with this transaction
}
