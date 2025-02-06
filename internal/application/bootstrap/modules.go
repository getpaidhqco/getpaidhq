package bootstrap

import (
	"go.uber.org/fx"
	"payloop/internal/api/controllers"
	"payloop/internal/api/middlewares"
	"payloop/internal/api/routes"
	"payloop/internal/application/services"
	"payloop/internal/infrastructure/authn/cognito"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/infrastructure/payments/paystack"
	"payloop/internal/infrastructure/pubsub/nats"
	"payloop/internal/infrastructure/workflow/temporal"
	"payloop/internal/lib"
)

var CommonModules = fx.Options(
	controllers.Module,
	routes.Module,
	lib.Module,
	services.Module,
	middlewares.Module,
	postgres.Module,

	// Authn
	cognito.Module,

	// Payment provider
	paystack.Module,

	// Workflow engine
	temporal.Module,

	// Pubsub
	nats.Module,
)
