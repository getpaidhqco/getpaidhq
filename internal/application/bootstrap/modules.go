package bootstrap

import (
	"go.uber.org/fx"
	"payloop/internal/api/controllers"
	"payloop/internal/api/middlewares"
	"payloop/internal/api/routes"
	"payloop/internal/application/services"
	"payloop/internal/domain/factories"
	"payloop/internal/infrastructure/authn/apikey"
	"payloop/internal/infrastructure/authn/clerk"
	"payloop/internal/infrastructure/authz/cedar"
	"payloop/internal/infrastructure/cache/redis"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/infrastructure/email/loops"
	"payloop/internal/infrastructure/payments/paystack"
	"payloop/internal/infrastructure/pubsub/nats"
	"payloop/internal/infrastructure/queue/sqs"
	"payloop/internal/infrastructure/scheduler/cron"
	"payloop/internal/infrastructure/storage/s3"
	"payloop/internal/infrastructure/vault/aes_vault"
	"payloop/internal/infrastructure/workflow/temporal"
	"payloop/internal/lib"
	"payloop/internal/mcp"
)

var CommonModules = fx.Options(
	controllers.Module,
	routes.Module,
	lib.Module,
	mcp.Module,
	services.Module,
	middlewares.Module,
	factories.Module,

	postgres.Module,

	// Security
	aes_vault.Module,

	// Authn & Authz
	//cognito.Module,
	clerk.Module,
	apikey.Module,
	cedar.Module,

	// Workflow engine
	temporal.Module,

	// Pubsub
	nats.Module,

	// Queue
	sqs.Module,

	// Cache client
	redis.Module,

	// Scheduler
	cron.Module,

	// Payment providers
	paystack.Module,

	// Email providers
	loops.Module,

	// File storage
	s3.Module(),
)
