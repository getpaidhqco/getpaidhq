package checkout_com

import "time"

type WebhookType string

const (
	PaymentCapturedWebhook WebhookType = "payment_captured"
	PaymentApprovedWebhook WebhookType = "payment_approved"
)

type WebhookData interface {
	GetID() string
}
type WebhookPayload struct {
	ID        string      `json:"id"`
	Type      WebhookType `json:"type"`
	Version   string      `json:"version"`
	CreatedOn time.Time   `json:"created_on"`
	Data      any
	Links     struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Subject struct {
			Href string `json:"href"`
		} `json:"subject"`
		Payment struct {
			Href string `json:"href"`
		} `json:"payment"`
		PaymentActions struct {
			Href string `json:"href"`
		} `json:"payment_actions"`
		Refund struct {
			Href string `json:"href"`
		} `json:"refund"`
	} `json:"_links"`
}

type PaymentCaptured struct {
	ID              string    `json:"id"`
	ActionID        string    `json:"action_id"`
	Reference       string    `json:"reference"`
	Amount          int       `json:"amount"`
	ProcessedOn     time.Time `json:"processed_on"`
	ResponseCode    string    `json:"response_code"`
	ResponseSummary string    `json:"response_summary"`
	Balances        struct {
		TotalAuthorized    int `json:"total_authorized"`
		TotalVoided        int `json:"total_voided"`
		AvailableToVoid    int `json:"available_to_void"`
		TotalCaptured      int `json:"total_captured"`
		AvailableToCapture int `json:"available_to_capture"`
		TotalRefunded      int `json:"total_refunded"`
		AvailableToRefund  int `json:"available_to_refund"`
	} `json:"balances"`
	Currency   string `json:"currency"`
	Processing struct {
		AcquirerTransactionID   string `json:"acquirer_transaction_id"`
		AcquirerReferenceNumber string `json:"acquirer_reference_number"`
	} `json:"processing"`
	EventLinks struct {
		Payment        string `json:"payment"`
		PaymentActions string `json:"payment_actions"`
		Refund         string `json:"refund"`
	} `json:"event_links"`
}

func (w PaymentCaptured) GetID() string {
	return w.ID
}

type PaymentApproved struct {
	ID        string `json:"id"`
	ActionID  string `json:"action_id"`
	Reference string `json:"reference"`
	Amount    int    `json:"amount"`
	AuthCode  string `json:"auth_code"`
	Currency  string `json:"currency"`
	Customer  struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"customer"`
	PaymentType string    `json:"payment_type"`
	ProcessedOn time.Time `json:"processed_on"`
	Metadata    struct {
		Phase               string `json:"phase"`
		CartID              string `json:"cart_id"`
		CorrelationID       string `json:"correlationId"`
		CkoContextID        string `json:"cko_context_id"`
		OrderID             string `json:"order_id"`
		OrgID               string `json:"org_id"`
		CkoPaymentSessionID string `json:"cko_payment_session_id"`
	} `json:"metadata"`
	Processing struct {
		AcquirerTransactionID    string `json:"acquirer_transaction_id"`
		RetrievalReferenceNumber string `json:"retrieval_reference_number"`
		Aft                      string `json:"aft"`
		Scheme                   string `json:"scheme"`
		SchemeMerchantID         string `json:"scheme_merchant_id"`
	} `json:"processing"`
	ResponseCode    string `json:"response_code"`
	ResponseSummary string `json:"response_summary"`
	Risk            struct {
		Flagged bool `json:"flagged"`
		Score   int  `json:"score"`
	} `json:"risk"`
	SchemeID string `json:"scheme_id"`
	Source   struct {
		ID             string `json:"id"`
		Type           string `json:"type"`
		BillingAddress struct {
			Line1    string `json:"line1"`
			Line2    string `json:"line2"`
			TownCity string `json:"town_city"`
			State    string `json:"state"`
			Zip      string `json:"zip"`
			Country  string `json:"country"`
		} `json:"billing_address"`
		ExpiryMonth   int    `json:"expiry_month"`
		ExpiryYear    int    `json:"expiry_year"`
		Name          string `json:"name"`
		Scheme        string `json:"scheme"`
		Last4         string `json:"last_4"`
		Fingerprint   string `json:"fingerprint"`
		Bin           string `json:"bin"`
		CardType      string `json:"card_type"`
		CardCategory  string `json:"card_category"`
		Issuer        string `json:"issuer"`
		IssuerCountry string `json:"issuer_country"`
		ProductType   string `json:"product_type"`
		AvsCheck      string `json:"avs_check"`
		CvvCheck      string `json:"cvv_check"`
	} `json:"source"`
	Balances struct {
		TotalAuthorized    int `json:"total_authorized"`
		TotalVoided        int `json:"total_voided"`
		AvailableToVoid    int `json:"available_to_void"`
		TotalCaptured      int `json:"total_captured"`
		AvailableToCapture int `json:"available_to_capture"`
		TotalRefunded      int `json:"total_refunded"`
		AvailableToRefund  int `json:"available_to_refund"`
	} `json:"balances"`
	EventLinks struct {
		Payment        string `json:"payment"`
		PaymentActions string `json:"payment_actions"`
		Capture        string `json:"capture"`
		Void           string `json:"void"`
	} `json:"event_links"`
	PanTypeProcessed         string `json:"pan_type_processed"`
	CkoNetworkTokenAvailable bool   `json:"cko_network_token_available"`
	PaymentIP                string `json:"payment_ip"`
}

func (w PaymentApproved) GetID() string {
	return w.ID
}

type InitPaymentOptions struct {
	Type string `json:"type"`
}
