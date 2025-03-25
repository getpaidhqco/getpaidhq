package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Customer struct {
	Id             string            `json:"id"`
	Name           string            `json:"name,omitempty"`
	Email          string            `json:"email"`
	FirstName      string            `json:"first_name,omitempty"`
	LastName       string            `json:"last_name,omitempty"`
	Phone          string            `json:"phone,omitempty"`
	BillingAddress entities.Address  `json:"billing_address,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at,omitempty"`
}

func NewCustomerFromEntity(entity entities.Customer) Customer {
	return Customer{
		Id:             entity.Id,
		Email:          entity.Email,
		FirstName:      entity.FirstName,
		LastName:       entity.LastName,
		BillingAddress: entity.BillingAddress,
		Metadata:       entity.Metadata,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
	}
}
