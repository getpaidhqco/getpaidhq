package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-fuego/fuego"
	"gorm.io/gorm"

	"getpaidhq/internal/adapter/cedar"
	"getpaidhq/internal/adapter/checkout_com"
	"getpaidhq/internal/adapter/clerk"
	"getpaidhq/internal/adapter/cron"
	"getpaidhq/internal/adapter/hatchet"
	hatchetsteps "getpaidhq/internal/adapter/hatchet/steps"
	handler "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/adapter/memory"
	"getpaidhq/internal/adapter/nats"
	"getpaidhq/internal/adapter/paystack"
	"getpaidhq/internal/adapter/postgres"
	"getpaidhq/internal/adapter/redis"
	"getpaidhq/internal/adapter/temporal"
	temporalact "getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// errUnsupportedEngine is returned when WORKFLOW_ENGINE is set to a
// value other than the recognized ones; previously this panicked
// inside NewApp which is hostile to anything trying to surface a
// startup config error cleanly.
func errUnsupportedEngine(name string) error {
	return fmt.Errorf("unsupported WORKFLOW_ENGINE %q (want 'hatchet' or 'temporal')", name)
}

// App holds all wired dependencies for the application.
type App struct {
	Server *fuego.Server
	DB     *gorm.DB
	Env    lib.Env
	// closers are long-lived resources (pubsub, workflow engine worker, cron
	// scheduler) torn down on shutdown, in reverse construction order.
	closers []io.Closer
}

// NewApp creates a new App with all dependencies manually wired.
func NewApp() (*App, error) {
	env := lib.NewEnv()
	logger := lib.GetLogger()
	reporter := lib.NewErrorReporter(logger)
	httpValidator := lib.NewValidator()

	// Parse trusted-proxy CIDRs once at boot. Malformed config fails the
	// app rather than silently degrading to "trust everything" — getting
	// this wrong is a security regression, not a runtime annoyance.
	trustedProxies, err := handler.ParseTrustedProxies(env.TrustedProxies)
	if err != nil {
		return nil, err
	}
	if len(trustedProxies) == 0 {
		logger.Warn("TRUSTED_PROXIES is empty — X-Forwarded-For / X-Real-IP will be ignored; UsedIp falls back to RemoteAddr")
	}

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

	txManager := postgres.NewTxManager(db)

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
	pubsub, err := nats.NewNatsPubSub(env.NatsURL, logger)
	if err != nil {
		return nil, err
	}
	cache := redis.NewRedisClient(env.Get("REDIS_HOST"), env.Get("REDIS_PASSWORD"), 0)
	authzEngine := cedar.NewCedarAuthz(logger, env)
	scheduler := cron.NewCronScheduler(logger, env)

	// Rate-limiter backend selection. With Redis configured the limit is
	// enforced cluster-wide (a shared GCRA budget across every instance);
	// without it we fall back to a per-instance in-memory bucket. The HTTP
	// middleware fails open if the backend errors, so this is never a hard
	// dependency.
	var rateLimiter port.RateLimiter
	if redisHost := env.Get("REDIS_HOST"); redisHost != "" {
		rateLimiter = redis.NewRateLimiter(redisHost, env.Get("REDIS_PASSWORD"), 0)
		logger.Info("Rate limiting backed by Redis (distributed, cluster-wide budget)")
	} else {
		rateLimiter = memory.NewRateLimiter(0)
		logger.Info("Rate limiting backed by in-memory store (per-instance budget; set REDIS_HOST for a cluster-wide limit)")
	}

	// Auth
	clerkAuth := clerk.NewClerkMiddleware(logger, env, metadataRepo)
	clerkProvider := clerk.NewClerkClient(env, logger, metadataRepo)
	authenticators := []port.Authenticator{clerkAuth}

	// Payment gateway adapters
	gatewayAdapters := map[domain.Gateway]port.GatewayAdapter{
		domain.Paystack:       paystack.NewAdapter(paymentRepo, pspRepo, settingRepo, logger, env.PaystackSecret),
		domain.CheckoutDotCom: checkout_com.NewAdapter(logger, env.CheckoutWebhookSecret),
	}
	gatewayFactory := service.NewGatewayFactory(pspRepo, settingRepo, logger, gatewayAdapters)

	_ = cache

	// ---------------------------------------------------------------------------
	// Narrow services (no workflow engine).
	// ---------------------------------------------------------------------------
	subService, err := service.NewSubscriptionService(sessionRepo, settingRepo, cartRepo, subRepo, customerRepo, orderRepo, paymentRepo, gatewayFactory, pubsub, reporter, logger, txManager)
	if err != nil {
		return nil, err
	}
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
		// Config error, not a runtime crash — surface to the caller so
		// the process exits cleanly with a non-zero status instead of
		// stack-dumping.
		return nil, errUnsupportedEngine(env.WorkflowEngine)
	}

	// Pubsub fan-in from subscription.* topics into the chosen engine. Lifted
	// out of the adapters so both share one implementation. Any subscribe
	// failure here fails the whole boot — see the de-panic note on each
	// constructor.
	if _, err := service.NewSubscriptionEventBridge(engine, pubsub, logger); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------------
	// Engine-aware services and the rest.
	// ---------------------------------------------------------------------------
	subOrchestrationService := service.NewSubscriptionOrchestrationService(subService, engine, logger)
	dunningOrchestrationService, err := service.NewDunningOrchestrationService(dunningService, dunningEngine, pubsub, reporter, logger)
	if err != nil {
		return nil, err
	}
	orderService := service.NewOrderService(txManager, engine, sessionRepo, priceRepo, cartRepo, orderRepo, customerRepo, subRepo, paymentRepo, paymentMethodRepo, productRepo, gatewayFactory, pubsub, logger)
	customerService, err := service.NewCustomerService(customerRepo, paymentMethodRepo, pubsub, logger, scheduler)
	if err != nil {
		return nil, err
	}
	productService := service.NewProductService(productRepo, variantRepo, priceRepo, cartRepo, logger, pubsub)
	sessionService := service.NewSessionService(sessionRepo, cartRepo, logger, pubsub)
	cartService := service.NewCartService(cartRepo, priceRepo, logger, productRepo)
	userService := service.NewUserService(userRepo)
	orgService := service.NewOrgService(orgRepo, pubsub, clerkProvider, customerRepo, settingRepo, metadataRepo, apiKeyRepo, logger, env.ApiKeyPepper)
	pspService := service.NewPspService(pspRepo, settingRepo, logger, pubsub)
	webhookService := service.NewWebhookService(logger, gatewayFactory, engine, idempotencyRepo, subRepo)
	reportService, err := service.NewReportService(logger, reportRepo, scheduler, orgRepo)
	if err != nil {
		return nil, err
	}
	if _, err := service.NewReportEventBridge(logger, pubsub, reportRepo); err != nil {
		return nil, err
	}
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
		Subscription:  handler.NewSubscriptionHandler(subOrchestrationService, logger, authzEngine),
		Customer:      customerHandler,
		Product:       handler.NewProductHandler(productService, logger, authzEngine),
		Cart:          handler.NewCartHandler(cartService, logger, authzEngine),
		Session:       handler.NewSessionHandler(sessionService, logger, authzEngine),
		Webhook:       handler.NewWebhookHandler(webhookService, logger),
		WebhookSub:    handler.NewWebhookSubscriptionHandler(webhookSubService, logger, authzEngine),
		Org:           handler.NewOrgHandler(orgService, logger),
		Report:        handler.NewReportHandler(reportService, logger),
		Psp:           handler.NewPspHandler(pspService, logger, authzEngine),
		PaymentMethod: handler.NewPaymentMethodHandler(customerHandler),
		Dunning:       handler.NewDunningHandler(dunningOrchestrationService, subService, logger, authzEngine, trustedProxies),
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
		RateLimiter:    rateLimiter,
	}, handlers)

	// Collect resources that own goroutines / connections so Run can tear them
	// down on shutdown. The workflow engine (Hatchet/Temporal worker) and cron
	// scheduler expose Close() via io.Closer; pubsub always does.
	closers := []io.Closer{pubsub}
	if c, ok := engine.(io.Closer); ok {
		closers = append(closers, c)
	}
	if c, ok := scheduler.(io.Closer); ok {
		closers = append(closers, c)
	}

	return &App{
		Server:  server,
		DB:      db,
		Env:     env,
		closers: closers,
	}, nil
}

// Run starts the HTTP server and blocks until it exits or an interrupt /
// SIGTERM arrives. fuego's Run() is a blocking http.Server.Serve with no
// signal handling of its own, so we run it in a goroutine and own the signal
// here: on shutdown we stop accepting connections (Server.Shutdown) and then
// tear down the workflow engine worker, pubsub and cron scheduler.
func (a *App) Run() error {
	logger := lib.GetLogger()

	srvErr := make(chan error, 1)
	go func() { srvErr <- a.Server.Run() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-srvErr:
		// Server exited on its own (e.g. failed to bind). Still tear down.
		a.shutdown(logger)
		return err
	case <-ctx.Done():
		stop() // restore default signal handling so a second signal hard-kills
		logger.Info("shutdown signal received, draining")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.Server.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown failed", "err", err.Error())
		}
		a.shutdown(logger)
		return nil
	}
}

// shutdown closes long-lived resources in reverse construction order. Best
// effort: every closer is attempted even if an earlier one errors.
func (a *App) shutdown(logger port.Logger) {
	for i := len(a.closers) - 1; i >= 0; i-- {
		if err := a.closers[i].Close(); err != nil {
			logger.Error("resource close failed during shutdown", "err", err.Error())
		}
	}
}
