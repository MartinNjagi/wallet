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
