package domain

import (
	"encoding/json"
	"time"
)

// PaymentMethod is a saved token / card-on-file scoped to a Customer.
type PaymentMethod struct {
	OrgId          string
	Id             string
	Status         PaymentMethodStatus
	Psp            string
	Name           string
	CustomerId     string
	BillingAddress Address
	Type           PaymentMethodType
	// Token is the PSP's reusable charge credential (Paystack
	// authorization_code, Checkout.com source id). Secret-typed: fmt/slog/
	// json all render "[redacted]", so pubsub events, outbound webhooks, and
	// any accidental logging or response DTO can't leak it. Reveal() only
	// where it leaves for the PSP (charge command construction) and at the
	// postgres column write.
	Token          Secret
	Details        any
	Metadata       map[string]string
	ExpireAt       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Address is a postal/billing address value object.
type Address struct {
	FirstName  string  `json:"first_name,omitempty"`
	LastName   string  `json:"last_name,omitempty"`
	Email      string  `json:"email,omitempty"`
	Phone      string  `json:"phone,omitempty"`
	Line1      string  `json:"line1,omitempty"`
	Line2      string  `json:"line2,omitempty"`
	City       string  `json:"city,omitempty"`
	State      string  `json:"state,omitempty"`
	PostalCode string  `json:"postal_code,omitempty"`
	Country    Country `json:"country,omitempty"`
}

func (a Address) IsEmpty() bool {
	return a.FirstName == "" &&
		a.LastName == "" &&
		a.Email == "" &&
		a.Phone == "" &&
		a.Line1 == "" &&
		a.Line2 == "" &&
		a.City == "" &&
		a.State == "" &&
		a.PostalCode == "" &&
		a.Country == ""
}

func ParseAddress(address map[string]any) Address {
	jsonData, _ := json.Marshal(address)
	var addr Address
	_ = json.Unmarshal(jsonData, &addr)
	return addr
}
