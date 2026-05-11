package paystack

import "time"

type WebhookPayload struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type TransactionSuccessful struct {
	ID              int64     `json:"id"`
	Domain          string    `json:"domain"`
	Status          string    `json:"status"`
	Reference       string    `json:"reference"`
	Amount          int64     `json:"amount"`
	Message         any       `json:"message"`
	GatewayResponse string    `json:"gateway_response"`
	PaidAt          time.Time `json:"paid_at"`
	CreatedAt       time.Time `json:"created_at"`
	Channel         string    `json:"channel"`
	Currency        string    `json:"currency"`
	IPAddress       string    `json:"ip_address"`

	// we dont specify a json tag here because we need to handle it as a special case
	// it's sometimes returned as a string and sometimes an object, so unmarhalling fails
	Metadata map[string]any `json:"-"`

	FeesBreakdown any `json:"fees_breakdown"`
	Log           any `json:"log"`
	Fees          int `json:"fees"`
	FeesSplit     any `json:"fees_split"`
	Authorization struct {
		AuthorizationCode         string `json:"authorization_code"`
		Bin                       string `json:"bin"`
		Last4                     string `json:"last4"`
		ExpMonth                  string `json:"exp_month"`
		ExpYear                   string `json:"exp_year"`
		Channel                   string `json:"channel"`
		CardType                  string `json:"card_type"`
		Bank                      string `json:"bank"`
		CountryCode               string `json:"country_code"`
		Brand                     string `json:"brand"`
		Reusable                  bool   `json:"reusable"`
		Signature                 string `json:"signature"`
		AccountName               any    `json:"account_name"`
		ReceiverBankAccountNumber any    `json:"receiver_bank_account_number"`
		ReceiverBank              any    `json:"receiver_bank"`
	} `json:"authorization"`
	Customer struct {
		ID                       int    `json:"id"`
		FirstName                string `json:"first_name"`
		LastName                 string `json:"last_name"`
		Email                    string `json:"email"`
		CustomerCode             string `json:"customer_code"`
		Phone                    string `json:"phone"`
		Metadata                 any    `json:"metadata"`
		RiskAction               string `json:"risk_action"`
		InternationalFormatPhone any    `json:"international_format_phone"`
	} `json:"customer"`
	Plan struct {
	} `json:"plan"`
	Subaccount struct {
	} `json:"subaccount"`
	Split struct {
	} `json:"split"`
	OrderID any `json:"order_id"`

	RequestedAmount    int `json:"requested_amount"`
	PosTransactionData any `json:"pos_transaction_data"`
	Source             struct {
		Type       string `json:"type"`
		Source     string `json:"source"`
		EntryPoint string `json:"entry_point"`
		Identifier any    `json:"identifier"`
	} `json:"source"`
}

type Metadata struct {
	CartID       string `json:"cart_id"`
	CustomFields []struct {
		DisplayName  string `json:"display_name"`
		VariableName string `json:"variable_name"`
		Value        string `json:"value"`
	} `json:"custom_fields"`
	OrderID string `json:"order_id"`
	Type    string `json:"type"`
	OrgID   string `json:"org_id"`
}

type RefundProcessed struct {
	Status               string `json:"status"`
	TransactionReference string `json:"transaction_reference"`
	RefundReference      any    `json:"refund_reference"`
	Amount               int64  `json:"amount"`
	Currency             string `json:"currency"`
	Customer             struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	} `json:"customer"`
	Integration  int64  `json:"integration"`
	Domain       string `json:"domain"`
	ID           string `json:"id"`
	CustomerNote string `json:"customer_note"`
	MerchantNote string `json:"merchant_note"`
}
