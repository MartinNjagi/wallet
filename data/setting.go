package data

type ClientConfiguration struct {
	ClientID               uint    `json:"client_id"`
	BaseSMSRate            float64 `json:"base_sms_rate"`
	RefundOnFailedDelivery bool    `json:"refund_on_failed_delivery"`
}
