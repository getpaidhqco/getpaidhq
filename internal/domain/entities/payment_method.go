package entities

import (
	"encoding/json"
	"payloop/internal/domain/common"
	"time"
)

type PaymentMethod struct {
	OrgId          string  `json:"org_id"`
	Id             string  `json:"id"`
	Psp            string  `json:"psp"`
	Name           string  `json:"name"`
	CustomerId     string  `json:"customer_id"`
	IsDefault      bool    `json:"is_default"`
	BillingAddress Address `json:"billing_address"`
	Type           string  `json:"type"`

	// TODO store this securely somewhere else
	Token string `json:"token"`

	Details   interface{}
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Address struct {
	FirstName  string         `json:"first_name"`
	LastName   string         `json:"last_name"`
	Email      string         `json:"email"`
	Phone      string         `json:"phone"`
	Line1      string         `json:"line1"`
	Line2      string         `json:"line2"`
	City       string         `json:"city"`
	State      string         `json:"state"`
	PostalCode string         `json:"postal_code"`
	Country    common.Country `json:"country"`
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
