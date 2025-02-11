package paystack

import "time"

type WebhookPayload struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type TransactionSuccessful struct {
	ID              int64       `json:"id"`
	Domain          string      `json:"domain"`
	Status          string      `json:"status"`
	Reference       string      `json:"reference"`
	Amount          int         `json:"amount"`
	Message         interface{} `json:"message"`
	GatewayResponse string      `json:"gateway_response"`
	PaidAt          time.Time   `json:"paid_at"`
	CreatedAt       time.Time   `json:"created_at"`
	Channel         string      `json:"channel"`
	Currency        string      `json:"currency"`
	IPAddress       string      `json:"ip_address"`
	Metadata        struct {
		CartID       string `json:"cart_id"`
		CustomFields []struct {
			DisplayName  string `json:"display_name"`
			VariableName string `json:"variable_name"`
			Value        string `json:"value"`
		} `json:"custom_fields"`
		OrderID string `json:"order_id"`
		OrgID   string `json:"org_id"`
	} `json:"metadata"`
	FeesBreakdown interface{} `json:"fees_breakdown"`
	Log           interface{} `json:"log"`
	Fees          int         `json:"fees"`
	FeesSplit     interface{} `json:"fees_split"`
	Authorization struct {
		AuthorizationCode         string      `json:"authorization_code"`
		Bin                       string      `json:"bin"`
		Last4                     string      `json:"last4"`
		ExpMonth                  string      `json:"exp_month"`
		ExpYear                   string      `json:"exp_year"`
		Channel                   string      `json:"channel"`
		CardType                  string      `json:"card_type"`
		Bank                      string      `json:"bank"`
		CountryCode               string      `json:"country_code"`
		Brand                     string      `json:"brand"`
		Reusable                  bool        `json:"reusable"`
		Signature                 string      `json:"signature"`
		AccountName               interface{} `json:"account_name"`
		ReceiverBankAccountNumber interface{} `json:"receiver_bank_account_number"`
		ReceiverBank              interface{} `json:"receiver_bank"`
	} `json:"authorization"`
	Customer struct {
		ID                       int         `json:"id"`
		FirstName                string      `json:"first_name"`
		LastName                 string      `json:"last_name"`
		Email                    string      `json:"email"`
		CustomerCode             string      `json:"customer_code"`
		Phone                    string      `json:"phone"`
		Metadata                 interface{} `json:"metadata"`
		RiskAction               string      `json:"risk_action"`
		InternationalFormatPhone interface{} `json:"international_format_phone"`
	} `json:"customer"`
	Plan struct {
	} `json:"plan"`
	Subaccount struct {
	} `json:"subaccount"`
	Split struct {
	} `json:"split"`
	OrderID interface{} `json:"order_id"`

	RequestedAmount    int         `json:"requested_amount"`
	PosTransactionData interface{} `json:"pos_transaction_data"`
	Source             struct {
		Type       string      `json:"type"`
		Source     string      `json:"source"`
		EntryPoint string      `json:"entry_point"`
		Identifier interface{} `json:"identifier"`
	} `json:"source"`
}
