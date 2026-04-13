package domain

import (
	"encoding/json"
	"time"
)

type PaymentMethod struct {
	OrgId          string              `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id             string              `gorm:"column:id;primaryKey" json:"id"`
	Status         PaymentMethodStatus `gorm:"column:status" json:"status"`
	Psp            string              `gorm:"column:psp" json:"psp"`
	Name           string              `gorm:"column:name" json:"name"`
	CustomerId     string              `gorm:"column:customer_id" json:"customer_id"`
	BillingAddress Address             `gorm:"column:billing_address;serializer:json" json:"billing_address,omitempty"`
	Type           PaymentMethodType   `gorm:"column:type" json:"type"`
	Token          string              `gorm:"column:token" json:"token"`
	Details        interface{}         `gorm:"column:details;serializer:json" json:"details,omitempty"`
	Metadata       map[string]string   `gorm:"column:metadata;serializer:json" json:"metadata,omitempty"`
	ExpireAt       time.Time           `gorm:"column:expire_at" json:"expire_at,omitempty"`
	CreatedAt      time.Time           `gorm:"column:created_at" json:"created_at,omitempty"`
	UpdatedAt      time.Time           `gorm:"column:updated_at" json:"updated_at,omitempty"`
}

func (PaymentMethod) TableName() string { return "payment_methods" }

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
