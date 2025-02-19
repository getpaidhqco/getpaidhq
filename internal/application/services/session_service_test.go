package services

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"payloop/internal/api/middlewares"
	"payloop/internal/domain/entities/carts"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/entities/sessions"
	"payloop/internal/domain/factories"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/lib"
	"testing"
)

func TestSessionService_CreateSession(t *testing.T) {
	ctx := context.Background()
	logger := lib.GetLogger()
	orgId := "mollie"
	request := sessions.CreateSessionInput{
		OrgId:    orgId,
		Currency: "ZAR",
		Country:  "ZA",
		Metadata: nil,
	}

	app := fx.New(fx.Options(
		lib.Module,
		Module,
		middlewares.Module,
		postgres.Module,
		factories.Module,
	), fx.Options(
		fx.Invoke(func(orderService OrderService, sessionService SessionService, cartService CartService) {

			session, err := sessionService.CreateSession(ctx, request)
			assert.Equal(t, err, nil)
			logger.Info("Session created", "session", session)

			_, err = cartService.AddProduct(ctx, carts.AddProductCommand{
				OrgId:     session.OrgId,
				CartId:    session.CartId,
				ProductId: "prod-1",
				PriceId:   "price-1",
				Quantity:  1,
			})
			assert.Equal(t, err, nil)

			order, _, err := orderService.CreateOrderFromCart(ctx, orders.CreateOrderInput{
				OrgId: orgId,
				Customer: orders.CreateOrderCommandCustomer{
					Name:  "John Doe",
					Email: "test@testie.com",
				},
				CartId:   session.CartId,
				Metadata: nil,
			})
			assert.Equal(t, err, nil)
			logger.Info("Order created", "order", order)
		}),
	))
	app.Start(ctx)
	defer func() {
		app.Stop(ctx)
	}()

}
