package data

type STKPushRequest struct {
	StkID         int64   `json:"stk_id"`
	Combination   string  `json:"combination"`
	Phone         uint64  `json:"phone,omitempty"`
	Amount        float64 `json:"amount,omitempty"`
	AccountRef    string  `json:"account_ref,omitempty"`
	ReferenceCode string  `json:"reference_code,omitempty"`
	SessionID     uint64  `json:"session_id,omitempty"`
	MerchantReqID string  `json:"merchant_req_id,omitempty"`
	CheckoutReqID string  `json:"checkout_req_id,omitempty"`
	Status        int     `json:"status,omitempty"`
}

type STKPushResponse struct {
	ReferenceCode string `json:"reference_code"`
	Message       string `json:"message"`
}

type STKCallbackPayload struct {
	CheckoutRequestID  string `json:"CheckoutRequestID"`
	ResultCode         int    `json:"ResultCode"`
	ResultDesc         string `json:"ResultDesc"`
	MpesaReceiptNumber string `json:"MpesaReceiptNumber"`
}

type AccessTokenPayload struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

type STKRequest struct {
	MSISDN      uint64  `json:"msisdn" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	SessionID   uint64  `json:"session_id" binding:"required"`
	Combination string  `json:"combination"`
}
