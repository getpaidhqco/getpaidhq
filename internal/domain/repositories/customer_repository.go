package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type CustomerRepository interface {
	FindById(ctx context.Context, id string) (entities.Customer, error)
	Create(ctx context.Context, entity entities.Customer) (entities.Customer, error)
}
