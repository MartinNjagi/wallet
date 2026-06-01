package models

import "gorm.io/gorm"

// MpesaTransaction tracks the lifecycle of an STK push top-up.
type MpesaTransaction struct {
	gorm.Model
	ClientID          uint
	CheckoutRequestID string  `gorm:"uniqueIndex"`
	Amount            float64 // Fiat amount
	Credits           int64   // Calculated credits based on their rate at the time
	Status            string  // "PENDING", "SUCCESS", "FAILED"
	ReceiptNumber     string  // M-Pesa receipt (e.g., RGT45...) populated on success
}

func (*MpesaTransaction) TableName() string {
	return "mpesa_transactions"
}
