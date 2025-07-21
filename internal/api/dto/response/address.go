package response

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
)

type Address struct {
	FirstName  string         `json:"first_name,omitempty"`
	LastName   string         `json:"last_name,omitempty"`
	Email      string         `json:"email,omitempty"`
	Phone      string         `json:"phone,omitempty"`
	Line1      string         `json:"line1,omitempty"`
	Line2      string         `json:"line2,omitempty"`
	City       string         `json:"city,omitempty"`
	State      string         `json:"state,omitempty"`
	PostalCode string         `json:"postal_code,omitempty"`
	Country    common.Country `json:"country,omitempty"`
}

// NewAddressFromEntity creates a response DTO from a domain entity
func NewAddressFromEntity(entity entities.Address) Address {
	return Address{
		FirstName:  entity.FirstName,
		LastName:   entity.LastName,
		Email:      entity.Email,
		Phone:      entity.Phone,
		Line1:      entity.Line1,
		Line2:      entity.Line2,
		City:       entity.City,
		State:      entity.State,
		PostalCode: entity.PostalCode,
		Country:    entity.Country,
	}
}