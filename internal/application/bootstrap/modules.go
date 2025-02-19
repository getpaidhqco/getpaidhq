package bootstrap

import (
	"go.uber.org/fx"
	"payloop/internal/api/controllers"
	"payloop/internal/api/middlewares"
	"payloop/internal/api/routes"
	"payloop/internal/application/services"
	"payloop/internal/domain/factories"
	"payloop/internal/infrastructure/authn/clerk"
	"payloop/internal/infrastructure/authz/cedar"
	"payloop/internal/infrastructure/db/postgres"
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
	factories.Module,
	
	postgres.Module,

	// Authn & Authz
	//cognito.Module,
	clerk.Module,
	cedar.Module,


	// Workflow engine
	temporal.Module,

	// Pubsub
	nats.Module,
)
