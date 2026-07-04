package models

import (
	"gorm.io/gorm"
	"time"
)

// MpesaTransaction tracks the lifecycle of an STK push top-up.
type MpesaTransaction struct {
	gorm.Model
	ClientID          uint
	SecureReference   string `gorm:"size:20;uniqueIndex"`  // The random code (used for C2B fallback)
	CheckoutRequestID string `gorm:"size:255;uniqueIndex"` // Used for STK callback
	Amount            float64
	Credits           int64
	Status            string // "PENDING", "SUCCESS", "FAILED"
	ReceiptNumber     string
}

func (*MpesaTransaction) TableName() string {
	return "mpesa_transactions"
}

// C2BTransaction tracks manual Paybill/Till payments.
type C2BTransaction struct {
	gorm.Model
	ClientID      uint   // Resolved from BillRefNumber
	TransactionID string `gorm:"size:255;uniqueIndex"` // M-Pesa receipt (e.g., RGT45...)
	Amount        float64
	Credits       int64
	BillRefNumber string // What the user typed as the account number
	MSISDN        string
	Status        string `gorm:"default:'SUCCESS'"`
}

func (*C2BTransaction) TableName() string {
	return "c2b_transactions"
}

// BankTransaction tracks manual wire transfers submitted for approval.
type BankTransaction struct {
	gorm.Model
	ClientID        uint
	Amount          float64
	Credits         int64
	ReferenceNumber string `gorm:"size:255;uniqueIndex"`
	ProofURL        string `gorm:"type:text"`
	Status          string `gorm:"default:'PENDING'"` // PENDING, APPROVED, REJECTED
	ApprovedBy      uint   // Admin ID who approved it
}

func (*BankTransaction) TableName() string {
	return "bank_transactions"
}

// STKPushRequest represents an M-PESA STK push transaction in the database
type STKPushRequest struct {
	ID            uint    `gorm:"primaryKey;autoIncrement;column:id"`
	MSISDN        uint64  `gorm:"not null;column:msisdn;index"`
	Amount        float64 `gorm:"not null;column:amount"`
	AccountRef    string  `gorm:"size:50;not null;column:account_ref"`
	ReferenceCode string  `gorm:"size:10;not null;column:reference_code;uniqueIndex"`
	SessionID     uint64  `gorm:"not null;column:session_id;index"`

	// M-PESA response fields
	MerchantRequestID string  `gorm:"size:100;column:merchant_request_id;index"`
	CheckoutRequestID string  `gorm:"size:100;column:checkout_request_id;index"`
	ResultCode        string  `gorm:"size:20;column:result_code"`
	ResultDescription string  `gorm:"type:text;column:result_description"`
	MpesaBalance      float64 `gorm:"not null;column:mpesa_balance"`

	// Status tracking
	Status      int    `gorm:"default:0;column:status;index"`
	Description string `gorm:"type:text;column:description"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at;index"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;column:updated_at"`
}

func (STKPushRequest) TableName() string {
	return "stk_push_requests"
}

// MpesaDispute tracks manual claims for missing M-Pesa transactions.
type MpesaDispute struct {
	gorm.Model
	ClientID      uint
	ReceiptNumber string `gorm:"size:255;uniqueIndex"` // e.g., RGT45MOCK
	MessageBody   string `gorm:"type:text"`            // The pasted SMS
	Amount        float64
	Status        string `gorm:"default:'PENDING'"` // PENDING, APPROVED, REJECTED
	ApprovedBy    uint
}

func (*MpesaDispute) TableName() string {
	return "mpesa_disputes"
}
