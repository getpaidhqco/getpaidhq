package services

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"payloop/internal/api/middlewares"
	"payloop/internal/infrastructure/authn/cognito"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/infrastructure/payments/paystack"
	"payloop/internal/infrastructure/pubsub/nats"
	"payloop/internal/lib"
	"testing"
)

func TestSubscriptionService_E2E(t *testing.T) {
	ctx := context.Background()
	orgId := "mollie"
	orderId := "ord_2sekVdNeZmszN7Eiz2sIjeoG9z4"

	app := fx.New(fx.Options(

		lib.Module,
		Module,
		middlewares.Module,
		postgres.Module,

		// Authn
		cognito.Module,

		// Payment provider
		paystack.Module,

		// Pubsub
		nats.Module,
	), fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			return lib.GetFxLogger()
		}),
		fx.Invoke(func(service SubscriptionService) {

			_, err := service.CreateSubscriptionsForOrder(ctx, orgId, orderId)
			assert.Equal(t, err, nil)
		}),
	))
	app.Start(ctx)
	defer func() {
		app.Stop(ctx)
	}()

}
