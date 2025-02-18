package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Customer struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewFromEntityCustomer(entity entities.Customer) Customer {
	return Customer{
		Id:        entity.Id,
		Name:      entity.Name,
		Email:     entity.Email,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}
