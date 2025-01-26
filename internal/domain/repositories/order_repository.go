package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type OrderRepository interface {
	FindByID(ctx context.Context, id uint) (*entities.Order, error)
	FindAll(ctx context.Context) ([]*entities.Order, error)
	Create(ctx context.Context, order entities.Order) error
	Update(ctx context.Context, order entities.Order) error
	Delete(ctx context.Context, id uint) error
}
