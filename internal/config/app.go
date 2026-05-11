package config

import (
	"payloop/internal/adapter/cedar"
	"payloop/internal/adapter/checkout_com"
	"payloop/internal/adapter/clerk"
	"payloop/internal/adapter/cron"
	handler "payloop/internal/adapter/http"
	"payloop/internal/adapter/nats"
	"payloop/internal/adapter/paystack"
	"payloop/internal/adapter/postgres"
	"payloop/internal/adapter/redis"
	"payloop/internal/adapter/sqs"
	"payloop/internal/adapter/temporal"
	"payloop/internal/adapter/temporal/activities"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/core/service"
	"payloop/internal/lib"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// App holds all wired dependencies for the application.
type App struct {
	Router *gin.Engine
	DB     *gorm.DB
	Env    lib.Env
}

// NewApp creates a new App with all dependencies manually wired.
func NewApp() (*App, error) {
	env := lib.NewEnv()
	logger := lib.GetLogger()
	reporter := lib.NewErrorReporter(logger)
	requestHandler := lib.NewRequestHandler(logger, reporter)

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

	// ---------------------------------------------------------------------------
	// Infrastructure adapters
	// ---------------------------------------------------------------------------
	pubsub := nats.NewNatsPubSub(logger)
	cache := redis.NewRedisClient(env.Get("REDIS_HOST"), env.Get("REDIS_PASSWORD"), 0)
	authzEngine := cedar.NewCedarAuthz(logger, env)
	scheduler := cron.NewCronScheduler(logger, env)
	queueClient := sqs.NewSQSFifoClient(logger, env)

	// Auth
	clerkAuth := clerk.NewClerkMiddleware(requestHandler, logger, env, metadataRepo)
	clerkProvider := clerk.NewClerkClient(env, logger, metadataRepo)
	authenticators := []port.Authenticator{clerkAuth}

	// Payment gateway adapters
	gatewayAdapters := map[domain.Gateway]port.GatewayAdapter{
		domain.Paystack:      paystack.NewAdapter(paymentRepo, pspRepo, settingRepo, logger),
		domain.CheckoutDotCom: checkout_com.NewAdapter(logger),
	}
	gatewayFactory := service.NewGatewayFactory(pspRepo, settingRepo, logger, gatewayAdapters)

	// Workflow engine holder: passed to services at construction so the real
	// Temporal engine can be plugged in once activities (which depend on
	// services) have been built.
	engineRef := &engineHolder{}
	var engine port.Engine = engineRef

	_ = cache
	_ = authenticators

	// ---------------------------------------------------------------------------
	// Services
	// ---------------------------------------------------------------------------
	subService := service.NewSubscriptionService(sessionRepo, settingRepo, cartRepo, subRepo, customerRepo, orderRepo, paymentRepo, pubsub, logger, engine)
	orderService := service.NewOrderService(engine, sessionRepo, priceRepo, cartRepo, orderRepo, customerRepo, subRepo, paymentRepo, paymentMethodRepo, productRepo, gatewayFactory, pubsub, logger)
	customerService := service.NewCustomerService(customerRepo, paymentMethodRepo, pubsub, logger, scheduler)
	productService := service.NewProductService(productRepo, variantRepo, priceRepo, cartRepo, logger, pubsub)
	sessionService := service.NewSessionService(sessionRepo, cartRepo, logger, pubsub)
	cartService := service.NewCartService(cartRepo, priceRepo, logger, productRepo)
	userService := service.NewUserService(userRepo)
	orgService := service.NewOrgService(orgRepo, pubsub, clerkProvider, customerRepo, settingRepo, metadataRepo, apiKeyRepo, logger)
	pspService := service.NewPspService(pspRepo, settingRepo, logger, pubsub)
	webhookService := service.NewWebhookService(logger, gatewayFactory, engine, idempotencyRepo, subRepo)
	webhookSubService := service.NewWebhookSubscriptionService(logger, webhookSubRepo, idempotencyRepo, pubsub)
	reportService := service.NewReportService(logger, reportRepo, pubsub, queueClient, nil, scheduler, orgRepo) // nil = cdc stream
	metadataService := service.NewMetadataService(metadataRepo, logger)

	// ---------------------------------------------------------------------------
	// Workflow engine (Temporal) — built after services so activities can hold
	// references to them, then plugged into engineRef so services see a live
	// engine at runtime.
	// ---------------------------------------------------------------------------
	orderActivities := activities.NewOrderActivities(orderService, settingRepo, subService, subRepo, pubsub, paymentRepo, gatewayFactory, reporter)
	webhookActivities := activities.NewOutgoingWebhookActivities(webhookSubRepo, settingRepo, webhookSubService, pubsub)
	engineRef.inner = temporal.NewTemporalEngine(logger, env, orderActivities, reporter, webhookActivities, settingRepo, pubsub)

	_ = metadataService
	_ = userService

	// ---------------------------------------------------------------------------
	// HTTP Handlers
	// ---------------------------------------------------------------------------
	healthHandler := handler.NewHealthHandler(logger)
	orderHandler := handler.NewOrderHandler(orderService, logger, authzEngine)
	subscriptionHandler := handler.NewSubscriptionHandler(subService, logger)
	customerHandler := handler.NewCustomerHandler(customerService, logger, authzEngine)
	productHandler := handler.NewProductHandler(productService, logger, authzEngine)
	cartHandler := handler.NewCartHandler(cartService, logger)
	sessionHandler := handler.NewSessionHandler(sessionService, logger, authzEngine)
	webhookHandler := handler.NewWebhookHandler(webhookService, logger)
	webhookSubHandler := handler.NewWebhookSubscriptionHandler(webhookSubService, logger, authzEngine)
	orgHandler := handler.NewOrgHandler(orgService, logger)
	reportHandler := handler.NewReportHandler(reportService, logger)
	pspHandler := handler.NewPspHandler(pspService, logger, authzEngine)
	paymentMethodHandler := handler.NewPaymentMethodHandler(customerHandler)

	// ---------------------------------------------------------------------------
	// HTTP Router
	// ---------------------------------------------------------------------------
	router := requestHandler.Gin

	api := router.Group("/api")
	healthHandler.RegisterRoutes(api)
	orderHandler.RegisterRoutes(api)
	subscriptionHandler.RegisterRoutes(api)
	customerHandler.RegisterRoutes(api)
	productHandler.RegisterRoutes(api)
	cartHandler.RegisterRoutes(api)
	sessionHandler.RegisterRoutes(api)
	webhookHandler.RegisterRoutes(api)
	webhookSubHandler.RegisterRoutes(api)
	orgHandler.RegisterRoutes(api)
	reportHandler.RegisterRoutes(api)
	pspHandler.RegisterRoutes(api)
	paymentMethodHandler.RegisterRoutes(api)

	return &App{
		Router: router,
		DB:     db,
		Env:    env,
	}, nil
}

// Run starts the HTTP server on the configured port.
func (a *App) Run() error {
	portNum := a.Env.ServerPort
	if portNum == "" {
		portNum = "8080"
	}
	return a.Router.Run(":" + portNum)
}
