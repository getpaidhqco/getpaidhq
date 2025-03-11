package services

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"payloop/internal/api/middlewares"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/infrastructure/db/postgres"
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
	), fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			return lib.GetFxLogger()
		}),
		fx.Invoke(func(orderService OrderService) {
			logger.Info("Starting application")

			_, err := orderService.CreateOrder(ctx, orders.CreateOrderInput{
				OrgId:    "org_2syb0uTnhuKtQTaLO6EAk1iIUnu",
				Currency: "ZAR",
				Customer: orders.CreateOrderCommandCustomer{
					Id: "cus_2u7124uRNWnn2NpQdpSa6b1kLqC",
				},
				PaymentMethodId: "pm_2u718M3todYa5mkGPM9JpCWWhw2",
				CartItems: []orders.CartItem{
					{ProductId: "prod-1", PriceId: "cyc-1", Quantity: 1},
				},
				PspId:    "Paystack",
				Metadata: nil,
			})
			assert.Equal(t, err, nil)
		}),
	))
	app.Start(ctx)
	defer func() {
		app.Stop(ctx)
	}()

}
