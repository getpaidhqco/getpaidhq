package request

import "payloop/internal/domain/common"

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
