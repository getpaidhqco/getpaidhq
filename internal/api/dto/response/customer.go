package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Customer struct {
	Id             string            `json:"id"`
	Name           string            `json:"name"`
	Email          string            `json:"email"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	Phone          string            `json:"phone"`
	BillingAddress entities.Address  `json:"billing_address"`
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
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
