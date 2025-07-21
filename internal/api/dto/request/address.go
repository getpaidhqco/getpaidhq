package request

import "payloop/internal/domain/common"

type Address struct {
	FirstName  string         `json:"first_name" binding:"omitempty,max=100"`
	LastName   string         `json:"last_name" binding:"omitempty,max=100"`
	Email      string         `json:"email" binding:"omitempty,email"`
	Phone      string         `json:"phone" binding:"omitempty,e164"`
	Line1      string         `json:"line1" binding:"omitempty,max=255"`
	Line2      string         `json:"line2" binding:"omitempty,max=255"`
	City       string         `json:"city" binding:"omitempty,max=100"`
	State      string         `json:"state" binding:"omitempty,max=100"`
	PostalCode string         `json:"postal_code" binding:"omitempty,max=20"`
	Country    common.Country `json:"country" binding:"omitempty"`
}

// IsEmpty checks if the address is empty
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
