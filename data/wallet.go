package data

// --- Requests ---

type InitiateTopUpRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Phone  string  `json:"phone" binding:"required"` // The phone number to prompt for STK
}

type ManualAdjustmentRequest struct {
	ClientID    uint   `json:"client_id" binding:"required"`
	Credits     int64  `json:"credits" binding:"required"`
	Action      string `json:"action" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type UpdateBillingConfigRequest struct {
	BaseSmsRate            *float64 `json:"base_sms_rate"`
	RefundOnFailedDelivery *bool    `json:"refund_on_failed_delivery"`
}

type WalletOperation struct {
	ClientID    uint
	Action      WalletAction
	Credits     int64 // always positive
	Type        string
	Description string
	Reference   string
	FiatPaid    *float64
	Currency    *string
}

// --- Bank Transfer Requests ---

type SubmitBankTransferRequest struct {
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	ReferenceNumber string  `json:"reference_number" binding:"required"`
	ProofURL        string  `json:"proof_url"` // Optional S3 link to uploaded receipt
}

type ApproveBankTransferRequest struct {
	Status      string `json:"status" binding:"required,oneof=APPROVED REJECTED"`
	Description string `json:"description"`
}

// --- Daraja C2B Payloads ---

type C2BValidationPayload struct {
	TransactionType   string `json:"TransactionType"`
	TransID           string `json:"TransID"`
	TransTime         string `json:"TransTime"`
	TransAmount       string `json:"TransAmount"`
	BusinessShortCode string `json:"BusinessShortCode"`
	BillRefNumber     string `json:"BillRefNumber"` // We map this to ClientID
	MSISDN            string `json:"MSISDN"`
	FirstName         string `json:"FirstName"`
}
