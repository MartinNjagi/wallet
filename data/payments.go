package data

// MpesaStatus – use integers for speed and compactness
const (
	MpesaStatusPending  = 0 // Awaiting callback
	MpesaStatusReceived = 1 // Received C2B/STK confirmation
	MpesaStatusLinked   = 2 // Linked to Ticket
	MpesaStatusFailed   = 3 // Failed / Reversed
)

// MpesaTransaction stores all C2B and STK transaction data
type MpesaTransaction struct {
	ReferenceID string  `json:"reference_id,omitempty"`
	TransTime   string  `json:"trans_time,omitempty"`
	TransAmount float64 `json:"trans_amount,omitempty"`
	MSISDN      uint64  `json:"msisdn,omitempty"`
	FirstName   string  `json:"first_name,omitempty"`
	MiddleName  string  `json:"middle_name,omitempty"`
	LastName    string  `json:"last_name,omitempty"`
	Status      int     `json:"status,omitempty"`
}

type C2BCallbackPayload struct {
	TransactionType   string `json:"TransactionType"`
	TransID           string `json:"TransID"`
	TransTime         string `json:"TransTime"`
	TransAmount       string `json:"TransAmount"`
	BusinessShortCode string `json:"BusinessShortCode"`
	BillRefNumber     string `json:"BillRefNumber"`
	InvoiceNumber     string `json:"InvoiceNumber"`
	OrgAccountBalance string `json:"OrgAccountBalance"`
	ThirdPartyTransID string `json:"ThirdPartyTransID"`
	MSISDN            string `json:"MSISDN"`
	FirstName         string `json:"FirstName"`
	MiddleName        string `json:"MiddleName"`
	LastName          string `json:"LastName"`
}

type MPESAExpressRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            string `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

type TransactionStatusRequest struct {
	Initiator                string `json:"Initiator"`
	SecurityCredential       string `json:"SecurityCredential"`
	CommandID                string `json:"CommandID"`
	TransactionID            string `json:"TransactionID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	PartyA                   string `json:"PartyA"`
	IdentifierType           string `json:"IdentifierType"`
	ResultURL                string `json:"ResultURL"`
	QueueTimeOutURL          string `json:"QueueTimeOutURL"`
	Remarks                  string `json:"Remarks"`
	Occasion                 string `json:"Occasion"`
}

// StkCallbackItem represents one item in the CallbackMetadata array
type StkCallbackItem struct {
	Name string `json:"Name"`
	// Value can be string (for receipt) or float64 (for balance/phone)
	Value interface{} `json:"Value"`
}

// CallbackMetadata houses the Item array
type CallbackMetadata struct {
	Item []StkCallbackItem `json:"Item"`
}

// StkCallback is the main payload body for the callback
type StkCallback struct {
	ResultCode int64  `json:"ResultCode"`
	ResultDesc string `json:"ResultDesc"`

	MerchantRequestID string `json:"MerchantRequestID"`
	CheckoutRequestID string `json:"CheckoutRequestID"`

	CallbackMetadata *CallbackMetadata `json:"CallbackMetadata,omitempty"` // Pointer because it's missing on failure
}

// StkCallbackPayload matches the full M-Pesa payload structure
type StkCallbackPayload struct {
	Body struct {
		StkCallback StkCallback `json:"stkCallback"`
	} `json:"Body"`
}

// TicketCreationContext holds all data needed for ticket creation
type TicketCreationContext struct {
	ProfileID     uint64
	PaymentID     uint
	MSISDN        uint64
	SessionID     int64
	ChannelID     uint
	Amount        float64
	ReferenceCode string
	Combination   string
	PaymentCode   string
}

type Payment struct {
	TransactionType   string `json:"TransactionType"`
	TransID           string `json:"TransID"`
	TransTime         string `json:"TransTime"`
	TransAmount       string `json:"TransAmount"`
	BusinessShortCode string `json:"BusinessShortCode"`
	BillRefNumber     string `json:"BillRefNumber"`
	InvoiceNumber     string `json:"InvoiceNumber"`
	OrgAccountBalance string `json:"OrgAccountBalance"`
	ThirdPartyTransID string `json:"ThirdPartyTransID"`
	MSISDN            string `json:"MSISDN"`
	FirstName         string `json:"FirstName"`
	MiddleName        string `json:"MiddleName"`
	LastName          string `json:"LastName"`
}

type TransactionStatusResult struct {
	Result struct {
		ResultType               int    `json:"ResultType"`
		ResultCode               int    `json:"ResultCode"`
		ResultDesc               string `json:"ResultDesc"`
		OriginatorConversationID string `json:"OriginatorConversationID"`
		ConversationID           string `json:"ConversationID"`
		TransactionID            string `json:"TransactionID"`
		ResultParameters         struct {
			ResultParameter []ResultParameter `json:"ResultParameter"`
		} `json:"ResultParameters"`
		ReferenceData struct {
			ReferenceItem struct {
				Key string `json:"Key"`
			} `json:"ReferenceItem"`
		} `json:"ReferenceData"`
	} `json:"Result"`
}

type ResultParameter struct {
	Key   string      `json:"Key"`
	Value interface{} `json:"Value"`
}

type TransactionStatusParams struct {
	DebitPartyName  string
	ReceiptNo       string
	Amount          string
	TransactionTime string
}

// TxStatusCallbackPayload Daraja Transaction Status Webhook Payload
type TxStatusCallbackPayload struct {
	Result struct {
		ResultType               int    `json:"ResultType"`
		ResultCode               int    `json:"ResultCode"`
		ResultDesc               string `json:"ResultDesc"`
		OriginatorConversationID string `json:"OriginatorConversationID"`
		ConversationID           string `json:"ConversationID"`
		TransactionID            string `json:"TransactionID"` // This is the Receipt Number
		ResultParameters         struct {
			ResultParameter []struct {
				Key   string      `json:"Key"`
				Value interface{} `json:"Value"`
			} `json:"ResultParameter"`
		} `json:"ResultParameters"`
	} `json:"Result"`
}
