package repositories

import (
	"context"
	"payloop/internal/domain/carts"
	"payloop/internal/domain/entities"
)

type CartRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Cart, error)
	Create(ctx context.Context, input carts.CreateCartInput) (entities.Cart, error)
	Update(ctx context.Context, input entities.Cart) (entities.Cart, error)
}
