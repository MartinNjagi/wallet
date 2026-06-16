package data

type BalanceCreditRequest struct {
	ClientID string `json:"client_id"`
}

type DeductCreditRequest struct {
	ClientID   uint   `json:"client_id"`
	Amount     uint   `json:"amount"`
	CampaignID string `json:"campaign_id"`
}

type RefundCreditRequest struct {
	ClientID   uint   `json:"client_id"`
	Amount     uint   `json:"amount"`
	CampaignID string `json:"campaign_id"`
}
