package checkout_com

type AuthenticationApproved struct {
	Eci                   string `json:"eci"`
	Cavv                  any    `json:"cavv"`
	Xid                   string `json:"xid"`
	Challenged            bool   `json:"challenged"`
	AcsChallengedMandated bool   `json:"acs_challenged_mandated"`
	ExemptionApplied      string `json:"exemption_applied"`
	Is3Ri                 bool   `json:"is3ri"`
	SessionID             string `json:"session_id"`
	Amount                string `json:"amount"`
	Currency              string `json:"currency"`
	Type                  string `json:"type"`
	ChallengeIndicator    string `json:"challenge_indicator"`
	ProtocolVersion       string `json:"protocol_version"`
	Scheme                string `json:"scheme"`
	PaymentID             string `json:"payment_id"`
	Reference             string `json:"reference"`
	ResponseCode          string `json:"response_code"`
	Experience            string `json:"experience"`
	ThreeDsTransactionID  string `json:"3ds_transaction_id"`
	DsTransactionID       string `json:"ds_transaction_id"`
	AcsTransactionID      string `json:"acs_transaction_id"`
}

func (w AuthenticationApproved) GetID() string {
	return w.Xid
}
