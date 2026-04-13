package domain

import (
	"encoding/json"
	"time"
)

type PaymentMethod struct {
	OrgId          string              `json:"org_id"`
	Id             string              `json:"id"`
	Status         PaymentMethodStatus `json:"status"`
	Psp            string              `json:"psp"`
	Name           string              `json:"name"`
	CustomerId     string              `json:"customer_id"`
	BillingAddress Address             `json:"billing_address,omitempty"`
	Type           PaymentMethodType   `json:"type"`
	Token          string              `json:"token"`
	Details        interface{}         `json:"details,omitempty"`
	Metadata       map[string]string   `json:"metadata,omitempty"`
	ExpireAt       time.Time           `json:"expire_at,omitempty"`
	CreatedAt      time.Time           `json:"created_at,omitempty"`
	UpdatedAt      time.Time           `json:"updated_at,omitempty"`
}

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

func ParseAddress(address map[string]interface{}) Address {
	jsonData, _ := json.Marshal(address)
	var addr Address
	_ = json.Unmarshal(jsonData, &addr)
	return addr
}
