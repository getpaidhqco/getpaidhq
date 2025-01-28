package services

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"payloop/internal/api/middlewares"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/infrastructure/payments/paystack"
	"payloop/internal/lib"
	"testing"
)

func TestCreateOrder(t *testing.T) {
	ctx := context.Background()
	logger := lib.GetLogger()

	app := fx.New(fx.Options(
		lib.Module,
		Module,
		middlewares.Module,
		postgres.Module,
		paystack.Module,
	), fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			return logger.GetFxLogger()
		}),
		fx.Invoke(func(orderService OrderService) {
			logger.Info("Starting application")

			_, err := orderService.CreateOrder(ctx, orders.CreateOrderCommand{
				CartId: "cart_id",
				OrgId:  "org_id",
				Customer: orders.CreateOrderCommandCustomer{
					Name:  "John Doe",
					Email: "test@payloop.com",
				},
			})
			assert.Equal(t, err, nil)
		}),
	))
	app.Start(ctx)
	defer func() {
		app.Stop(ctx)
	}()

}
