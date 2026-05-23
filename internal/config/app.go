package config

import (
	"github.com/go-fuego/fuego"
	"gorm.io/gorm"

	"getpaidhq/internal/adapter/cedar"
	"getpaidhq/internal/adapter/checkout_com"
	"getpaidhq/internal/adapter/clerk"
	"getpaidhq/internal/adapter/cron"
	"getpaidhq/internal/adapter/hatchet"
	hatchetsteps "getpaidhq/internal/adapter/hatchet/steps"
	handler "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/adapter/nats"
	"getpaidhq/internal/adapter/paystack"
	"getpaidhq/internal/adapter/postgres"
	"getpaidhq/internal/adapter/redis"
	"getpaidhq/internal/adapter/sqs"
	"getpaidhq/internal/adapter/temporal"
	temporalact "getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// App holds all wired dependencies for the application.
type App struct {
	Server *fuego.Server
	DB     *gorm.DB
	Env    lib.Env
}

// NewApp creates a new App with all dependencies manually wired.
func NewApp() (*App, error) {
	env := lib.NewEnv()
	logger := lib.GetLogger()
	reporter := lib.NewErrorReporter(logger)
	httpValidator := lib.NewValidator(logger)

	// ---------------------------------------------------------------------------
	// Database
	// ---------------------------------------------------------------------------
	db, err := postgres.NewDatabase(env.Get("DATABASE_URL"), logger)
	if err != nil {
		return nil, err
	}
	reportDB, err := postgres.NewDatabase(env.Get("REPORTING_DATABASE_URL"), logger)
	if err != nil {
		reportDB = db
	}

	// ---------------------------------------------------------------------------
	// Repositories
	// ---------------------------------------------------------------------------
	subRepo := postgres.NewSubscriptionRepo(db)
	orderRepo := postgres.NewOrderRepo(db)
	customerRepo := postgres.NewCustomerRepo(db)
	paymentRepo := postgres.NewPaymentRepo(db)
	paymentMethodRepo := postgres.NewPaymentMethodRepo(db)
	productRepo := postgres.NewProductRepo(db)
	variantRepo := postgres.NewVariantRepo(db)
	priceRepo := postgres.NewPriceRepo(db)
	sessionRepo := postgres.NewSessionRepo(db)
	cartRepo := postgres.NewCartRepo(db)
	orgRepo := postgres.NewOrgRepo(db)
	userRepo := postgres.NewUserRepo(db)
	settingRepo := postgres.NewSettingRepo(db)
	webhookSubRepo := postgres.NewWebhookSubscriptionRepo(db)
	apiKeyRepo := postgres.NewApiKeyRepo(db)
	idempotencyRepo := postgres.NewIdempotencyKeyRepo(db)
	pspRepo := postgres.NewPspRepo(db)
	metadataRepo := postgres.NewMetadataStoreRepo(db)
	reportRepo := postgres.NewReportRepo(reportDB)
	dunningRepo := postgres.NewDunningRepo(db)

	// ---------------------------------------------------------------------------
	// Infrastructure adapters
	// ---------------------------------------------------------------------------
	pubsub := nats.NewNatsPubSub(logger)
	cache := redis.NewRedisClient(env.Get("REDIS_HOST"), env.Get("REDIS_PASSWORD"), 0)
	authzEngine := cedar.NewCedarAuthz(logger, env)
	scheduler := cron.NewCronScheduler(logger, env)
	queueClient, err := sqs.NewSQSFifoClient(logger, env)
	if err != nil {
		return nil, err
	}

	// Auth
	clerkAuth := clerk.NewClerkMiddleware(logger, env, metadataRepo)
	clerkProvider := clerk.NewClerkClient(env, logger, metadataRepo)
	authenticators := []port.Authenticator{clerkAuth}

	// Payment gateway adapters
	gatewayAdapters := map[domain.Gateway]port.GatewayAdapter{
		domain.Paystack:       paystack.NewAdapter(paymentRepo, pspRepo, settingRepo, logger),
		domain.CheckoutDotCom: checkout_com.NewAdapter(logger),
	}
	gatewayFactory := service.NewGatewayFactory(pspRepo, settingRepo, logger, gatewayAdapters)

	_ = cache

	// ---------------------------------------------------------------------------
	// Narrow services (no workflow engine).
	// ---------------------------------------------------------------------------
	subService := service.NewSubscriptionService(sessionRepo, settingRepo, cartRepo, subRepo, customerRepo, orderRepo, paymentRepo, gatewayFactory, pubsub, reporter, logger)
	paymentService := service.NewPaymentService(paymentRepo, logger)
	orderWorkflowService := service.NewOrderWorkflowService(orderRepo, customerRepo, subRepo, paymentMethodRepo, paymentRepo, pubsub, logger)
	dunningService := service.NewDunningService(dunningRepo, subRepo, customerRepo, paymentRepo, subService, gatewayFactory, pubsub, reporter, logger)

	webhookSubService := service.NewWebhookSubscriptionService(logger, webhookSubRepo, idempotencyRepo, pubsub)

	// ---------------------------------------------------------------------------
	// Workflow engine selection — WORKFLOW_ENGINE env var picks the adapter.
	// Defaults to hatchet (see lib.NewEnv()).
	//
	// Both adapters take engine-agnostic services in their constructors; the
	// Hatchet/Temporal-specific shim layer (steps vs activities) is built per
	// engine so the same business logic is reused unchanged.
	// ---------------------------------------------------------------------------
	var engine port.Engine
	var dunningEngine port.DunningEngine
	switch env.WorkflowEngine {
	case "temporal":
		orderActivities := temporalact.NewOrderActivities(orderWorkflowService, subService, paymentService, subRepo)
		webhookActivities := temporalact.NewOutgoingWebhookActivities(webhookSubService)
		dunningActivities := temporalact.NewDunningActivities(dunningService)
		t := temporal.NewTemporalEngine(logger, env, orderActivities, webhookActivities, dunningActivities, reporter)
		engine = t
		dunningEngine = t
	case "hatchet", "":
		webhookSteps := hatchetsteps.NewOutgoingWebhookSteps(logger, webhookSubRepo, settingRepo, webhookSubService, pubsub)
		dunningSteps := hatchetsteps.NewDunningSteps(logger, dunningService)
		h := hatchet.NewHatchetEngine(logger, env, orderWorkflowService, subService, paymentService, subRepo, reporter, webhookSteps, dunningSteps)
		engine = h
		dunningEngine = h
	default:
		logger.Errorf("Unsupported WORKFLOW_ENGINE=%q; must be 'hatchet' or 'temporal'", env.WorkflowEngine)
		panic("unsupported WORKFLOW_ENGINE: " + env.WorkflowEngine)
	}

	// Pubsub fan-in from subscription.* topics into the chosen engine. Lifted
	// out of the adapters so both share one implementation.
	_ = service.NewSubscriptionEventBridge(engine, pubsub, logger)

	// ---------------------------------------------------------------------------
	// Engine-aware services and the rest.
	// ---------------------------------------------------------------------------
	subOrchestrationService := service.NewSubscriptionOrchestrationService(subService, engine, logger)
	dunningOrchestrationService := service.NewDunningOrchestrationService(dunningService, dunningEngine, pubsub, reporter, logger)
	orderService := service.NewOrderService(engine, sessionRepo, priceRepo, cartRepo, orderRepo, customerRepo, subRepo, paymentRepo, paymentMethodRepo, productRepo, gatewayFactory, pubsub, logger)
	customerService := service.NewCustomerService(customerRepo, paymentMethodRepo, pubsub, logger, scheduler)
	productService := service.NewProductService(productRepo, variantRepo, priceRepo, cartRepo, logger, pubsub)
	sessionService := service.NewSessionService(sessionRepo, cartRepo, logger, pubsub)
	cartService := service.NewCartService(cartRepo, priceRepo, logger, productRepo)
	userService := service.NewUserService(userRepo)
	orgService := service.NewOrgService(orgRepo, pubsub, clerkProvider, customerRepo, settingRepo, metadataRepo, apiKeyRepo, logger)
	pspService := service.NewPspService(pspRepo, settingRepo, logger, pubsub)
	webhookService := service.NewWebhookService(logger, gatewayFactory, engine, idempotencyRepo, subRepo)
	reportService := service.NewReportService(logger, reportRepo, pubsub, queueClient, scheduler, orgRepo)
	metadataService := service.NewMetadataService(metadataRepo, logger)

	_ = metadataService
	_ = userService

	// ---------------------------------------------------------------------------
	// HTTP Handlers
	// ---------------------------------------------------------------------------
	customerHandler := handler.NewCustomerHandler(customerService, logger, authzEngine)
	handlers := Handlers{
		Health:        handler.NewHealthHandler(logger),
		Order:         handler.NewOrderHandler(orderService, logger, authzEngine),
		Subscription:  handler.NewSubscriptionHandler(subOrchestrationService, logger),
		Customer:      customerHandler,
		Product:       handler.NewProductHandler(productService, logger, authzEngine),
		Cart:          handler.NewCartHandler(cartService, logger),
		Session:       handler.NewSessionHandler(sessionService, logger, authzEngine),
		Webhook:       handler.NewWebhookHandler(webhookService, logger),
		WebhookSub:    handler.NewWebhookSubscriptionHandler(webhookSubService, logger, authzEngine),
		Org:           handler.NewOrgHandler(orgService, logger),
		Report:        handler.NewReportHandler(reportService, logger),
		Psp:           handler.NewPspHandler(pspService, logger, authzEngine),
		PaymentMethod: handler.NewPaymentMethodHandler(customerHandler),
		Dunning:       handler.NewDunningHandler(dunningOrchestrationService, subService, logger, authzEngine),
	}

	port := env.ServerPort
	if port == "" {
		port = "8080"
	}
	server := BuildServer(ServerDeps{
		Addr:           ":" + port,
		Logger:         logger,
		Validator:      httpValidator,
		Authenticators: authenticators,
		Env:            env,
	}, handlers)

	return &App{
		Server: server,
		DB:     db,
		Env:    env,
	}, nil
}

// Run starts the HTTP server.
func (a *App) Run() error {
	return a.Server.Run()
}
