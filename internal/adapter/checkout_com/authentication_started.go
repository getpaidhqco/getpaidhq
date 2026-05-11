package checkout_com

type AuthenticationStarted struct {
	SessionID            string `json:"session_id"`
	Amount               string `json:"amount"`
	Currency             string `json:"currency"`
	Type                 string `json:"type"`
	ChallengeIndicator   string `json:"challenge_indicator"`
	ProtocolVersion      string `json:"protocol_version"`
	Scheme               string `json:"scheme"`
	PaymentID            string `json:"payment_id"`
	Reference            string `json:"reference"`
	ResponseCode         any    `json:"response_code"`
	Experience           any    `json:"experience"`
	ThreeDsTransactionID any    `json:"3ds_transaction_id"`
	DsTransactionID      any    `json:"ds_transaction_id"`
	AcsTransactionID     any    `json:"acs_transaction_id"`
}

func (w AuthenticationStarted) GetID() string {
	return w.SessionID
}
