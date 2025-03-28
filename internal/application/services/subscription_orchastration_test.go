package services

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"payloop/internal/api/middlewares"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/factories"
	"payloop/internal/infrastructure/authn/apikey"
	"payloop/internal/infrastructure/authn/cognito"
	"payloop/internal/infrastructure/authz/cedar"
	"payloop/internal/infrastructure/cache/redis"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/infrastructure/pubsub/nats"
	"payloop/internal/infrastructure/queue/sqs"
	"payloop/internal/infrastructure/scheduler/cron"
	"payloop/internal/infrastructure/workflow/temporal"
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

func TestSubscriptionOrchestrationService_UpdateWorkflowState(t *testing.T) {
	ctx := context.Background()
	orgId := "mollie"
	subId := "sub_2uwLeqA4zpSpBDY66jiajHpG0A6"

	app := fx.New(fx.Options(
		lib.Module,
		Module,
		middlewares.Module,
		factories.Module,

		postgres.Module,

		// Authn & Authz
		//cognito.Module,
		//clerk.Module,
		apikey.Module,
		cedar.Module,

		// Pubsub
		nats.Module,

		// Queue
		sqs.Module,

		// Cache client
		redis.Module,

		// Scheduler
		cron.Module,

		// Workflow
		temporal.Module,
	), fx.Options(
		fx.Invoke(func(service interfaces.SubscriptionOrchestrationService) {
			_, err := service.UpdateWorkflowState(ctx, orgId, subId)
			assert.Equal(t, err, nil)
		}),
	))
	app.Start(ctx)
	defer func() {
		app.Stop(ctx)
	}()

}
