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
	"github.com/nats-io/nats.go/jetstream"

	"getpaidhq/internal/adapter/apikey"
	"getpaidhq/internal/adapter/cedar"
	"getpaidhq/internal/adapter/checkout_com"
	"getpaidhq/internal/adapter/clerk"
	"getpaidhq/internal/adapter/cron"
	"getpaidhq/internal/adapter/crypto"
	"getpaidhq/internal/adapter/hatchet"
	hatchetsteps "getpaidhq/internal/adapter/hatchet/steps"
	handler "getpaidhq/internal/adapter/http"
	gphqjetstream "getpaidhq/internal/adapter/jetstream"
	"getpaidhq/internal/adapter/memory"
	"getpaidhq/internal/adapter/nats"
	"getpaidhq/internal/adapter/paystack"
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

// buildIngestor selects the durable write path from USAGE_INGEST_MODE:
//   - "sync" (default): the EventStore itself — a direct write on the request path.
//   - "jetstream": publish events durably to NATS JetStream; a background consumer
//     drains into the EventStore. Returns the ingestor and, for jetstream, the
//     consumer worker as an io.Closer the caller must register (so it stops before
//     the shared NATS connection drains).
func buildIngestor(env lib.Env, store port.EventStore, pubsub port.PubSub, logger lib.Logger) (port.EventIngestor, io.Closer, error) {
	switch env.UsageIngestMode {
	case "", "sync":
		return store, nil, nil
	case "jetstream":
		np, ok := pubsub.(*nats.NatsPubSub)
		if !ok || np.Conn() == nil {
			return nil, nil, fmt.Errorf("USAGE_INGEST_MODE=jetstream requires the NATS pubsub adapter with a live connection")
		}
		js, err := jetstream.New(np.Conn())
		if err != nil {
			return nil, nil, fmt.Errorf("jetstream context: %w", err)
		}
		consumer, err := gphqjetstream.NewConsumer(context.Background(), store, js, env.UsageIngestBatchSize, logger)
		if err != nil {
			return nil, nil, fmt.Errorf("start jetstream usage consumer: %w", err)
		}
		return gphqjetstream.NewIngestor(js, logger), consumer, nil
	default:
		return nil, nil, fmt.Errorf("unsupported USAGE_INGEST_MODE %q (want 'sync' or 'jetstream')", env.UsageIngestMode)
	}
}

// App holds all wired dependencies for the application.
type App struct {
	Server *fuego.Server
	// DB is the raw operational handle (*gorm.DB or *pgxpool.Pool, per DB_DRIVER),
	// retained for diagnostics/health; the app talks to storage through ports.
	DB  any
	Env lib.Env
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
	// Storage adapter (gorm | pgx, selected by DB_DRIVER) — every repo, the tx
	// manager, the event store and the prior-payment checker come from one
	// driver-specific builder. Downstream code depends only on the ports.
	//
	// Reporting persistence is intentionally not wired (see report_repo.go).
	// ---------------------------------------------------------------------------
	repos, err := newRepoSet(env, logger)
	if err != nil {
		return nil, err
	}
	db := repos.operationalDB
	txManager := repos.tx
	subRepo := repos.subscription
	orderRepo := repos.order
	customerRepo := repos.customer
	paymentRepo := repos.payment
	paymentMethodRepo := repos.paymentMethod
	productRepo := repos.product
	variantRepo := repos.variant
	priceRepo := repos.price
	sessionRepo := repos.session
	cartRepo := repos.cart
	orgRepo := repos.org
	userRepo := repos.user
	settingRepo := repos.setting
	webhookSubRepo := repos.webhookSub
	apiKeyRepo := repos.apiKey
	idempotencyRepo := repos.idempotency
	pspRepo := repos.psp
	metadataRepo := repos.metadata
	dunningRepo := repos.dunning
	invoiceRepo := repos.invoice
	meterRepo := repos.meter
	couponRepo := repos.coupon
	couponCodeRepo := repos.couponCode
	discountRepo := repos.discount
	priorPaymentChecker := repos.priorPaymentChecker
	eventStore := repos.eventStore

	// ---------------------------------------------------------------------------
	// Infrastructure adapters
	// ---------------------------------------------------------------------------
	pubsub, err := nats.NewNatsPubSub(env.NatsURL, logger)
	if err != nil {
		return nil, err
	}
	cache := redis.NewRedisClient(env.Get("REDIS_HOST"), env.Get("REDIS_PASSWORD"), 0)
	authzEngine := cedar.NewCedarAuthz(logger, env.CedarPolicyFile)
	scheduler := cron.NewCronScheduler(logger)

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
	clerkAuth := clerk.NewClerkMiddleware(logger, env.ClerkSecretKey, metadataRepo)
	clerkProvider := clerk.NewClerkClient(env.ClerkSecretKey, logger, metadataRepo)
	// Clerk is tried first; a non-Clerk token (an x-api-key value) fails its
	// check and falls through to the API-key authenticator. Order matters only
	// for which one "wins" a token it can actually validate, and the two token
	// shapes are disjoint.
	apiKeyAuth := apikey.NewApiKeyMiddleware(logger, env.ApiKeyPepper, apiKeyRepo)
	authenticators := []port.Authenticator{clerkAuth, apiKeyAuth}

	// Cipher for stored PSP credentials. Optional at boot: with no
	// SECRETS_ENCRYPTION_KEY the cipher is nil and configuring/using a
	// gateway fails with a clear error at that point instead of the whole
	// server refusing to start. A key that is SET but invalid is a config
	// bug and does fail boot.
	var secretCipher port.SecretCipher
	if env.SecretsEncryptionKey != "" {
		cipher, err := crypto.NewAesGcmCipher(env.SecretsEncryptionKey)
		if err != nil {
			return nil, err
		}
		secretCipher = cipher
	}

	// Payment gateway adapters
	gatewayAdapters := map[domain.Gateway]port.GatewayAdapter{
		domain.Paystack:       paystack.NewAdapter(paymentRepo, pspRepo, secretCipher, logger, env.PaystackSecret),
		domain.CheckoutDotCom: checkout_com.NewAdapter(logger, env.CheckoutWebhookSecret),
		// In-memory, always-succeeds gateway. Harmless in prod (only used if an
		// org's PSP config selects "memory"); enables local/offline charge testing.
		domain.Memory: memory.NewGatewayAdapter(logger),
	}
	gatewayFactory := service.NewGatewayFactory(pspRepo, secretCipher, logger, gatewayAdapters)

	_ = cache

	// ---------------------------------------------------------------------------
	// Narrow services (no workflow engine).
	// ---------------------------------------------------------------------------
	ingestor, ingestCloser, err := buildIngestor(env, eventStore, pubsub, logger)
	if err != nil {
		return nil, err
	}
	usageService := service.NewUsageService(meterRepo, customerRepo, subRepo, orderRepo, priceRepo, ingestor, eventStore, pubsub, logger)
	meterService := service.NewMeterService(meterRepo, pubsub, logger)
	invoiceService := service.NewInvoiceService(invoiceRepo, orderRepo, priceRepo, usageService, txManager, logger)
	couponService := service.NewCouponService(couponRepo, couponCodeRepo, discountRepo, priorPaymentChecker, txManager, logger)
	subService, err := service.NewSubscriptionService(sessionRepo, settingRepo, cartRepo, subRepo, customerRepo, orderRepo, paymentRepo, priceRepo, gatewayFactory, invoiceService, pubsub, reporter, logger, txManager)
	if err != nil {
		return nil, err
	}
	paymentService := service.NewPaymentService(paymentRepo, logger)
	orderWorkflowService := service.NewOrderWorkflowService(orderRepo, customerRepo, subRepo, paymentMethodRepo, paymentRepo, priceRepo, pubsub, logger)
	dunningService := service.NewDunningService(dunningRepo, subRepo, customerRepo, paymentRepo, subService, gatewayFactory, pubsub, reporter, logger)

	webhookSubService := service.NewWebhookSubscriptionService(logger, webhookSubRepo, idempotencyRepo, pubsub)

	// Built before the engine wiring because the Hatchet engine takes this as the
	// per-tenant reminder-config resolver (port.ReminderConfigResolver) for the
	// billing-sweep fan-out. Also consumed by the reminder-config HTTP handler below.
	reminderConfigService := service.NewReminderConfigService(settingRepo, logger)

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
		orderActivities := temporalact.NewOrderActivities(orderWorkflowService, subService, paymentService, subRepo, reminderConfigService)
		webhookActivities := temporalact.NewOutgoingWebhookActivities(webhookSubService)
		dunningActivities := temporalact.NewDunningActivities(dunningService)
		t := temporal.NewTemporalEngine(logger, temporal.Config{
			HostPort:  env.TemporalHost,
			Namespace: env.TemporalNamespace,
			TaskQueue: env.TemporalTaskQueue,
		}, orderActivities, webhookActivities, dunningActivities, reporter)
		engine = t
		dunningEngine = t
	case "hatchet", "":
		webhookSteps := hatchetsteps.NewOutgoingWebhookSteps(logger, webhookSubRepo, settingRepo, webhookSubService, pubsub)
		dunningSteps := hatchetsteps.NewDunningSteps(logger, dunningService)
		h := hatchet.NewHatchetEngine(logger, hatchet.Config{
			HostPort:             env.HatchetHostPort,
			Namespace:            env.HatchetNamespace,
			BillingSweepInterval: env.HatchetBillingSweepInterval,
			LogLevel:             env.HatchetLogLevel,
			TracingEnabled:       env.HatchetTracingEnabled,
		}, orderWorkflowService, subService, paymentService, subRepo, orgRepo, reminderConfigService, webhookSteps, dunningSteps)
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
	apiKeyService := service.NewApiKeyService(apiKeyRepo, env.ApiKeyPepper, logger)
	settingService := service.NewSettingService(settingRepo, logger)
	pspService := service.NewPspService(pspRepo, secretCipher, logger, pubsub)
	webhookService := service.NewWebhookService(logger, gatewayFactory, engine, idempotencyRepo, subRepo)
	metadataService := service.NewMetadataService(metadataRepo, logger)

	_ = metadataService
	_ = userService

	// ---------------------------------------------------------------------------
	// HTTP Handlers
	// ---------------------------------------------------------------------------
	customerHandler := handler.NewCustomerHandler(customerService, logger, authzEngine)
	handlers := Handlers{
		Health:         handler.NewHealthHandler(logger),
		Order:          handler.NewOrderHandler(orderService, logger, authzEngine),
		Subscription:   handler.NewSubscriptionHandler(subOrchestrationService, logger, authzEngine),
		Customer:       customerHandler,
		Product:        handler.NewProductHandler(productService, logger, authzEngine),
		Cart:           handler.NewCartHandler(cartService, logger, authzEngine),
		Session:        handler.NewSessionHandler(sessionService, logger, authzEngine),
		Webhook:        handler.NewWebhookHandler(webhookService, logger),
		WebhookSub:     handler.NewWebhookSubscriptionHandler(webhookSubService, logger, authzEngine),
		Org:            handler.NewOrgHandler(orgService, logger),
		Psp:            handler.NewPspHandler(pspService, logger, authzEngine),
		PaymentMethod:  handler.NewPaymentMethodHandler(customerService),
		Dunning:        handler.NewDunningHandler(dunningOrchestrationService, subService, logger, authzEngine, trustedProxies),
		ApiKey:         handler.NewApiKeyHandler(apiKeyService, logger, authzEngine),
		ReminderConfig: handler.NewReminderConfigHandler(reminderConfigService, logger),
		Usage:          handler.NewUsageHandler(usageService, logger, authzEngine),
		Meter:          handler.NewMeterHandler(meterService, logger, authzEngine),
		Invoice:        handler.NewInvoiceHandler(invoiceService, logger, authzEngine),
		Coupon:         handler.NewCouponHandler(couponService, logger, authzEngine),
		Payment:        handler.NewPaymentHandler(paymentService, logger, authzEngine),
		Setting:        handler.NewSettingHandler(settingService, logger, authzEngine),
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
		AllowedOrigins: env.AllowedOrigins,
		TrustedProxies: env.TrustedProxies,
		RateLimitRPS:   env.RateLimitRPS,
		RateLimitBurst: env.RateLimitBurst,
		RateLimiter:    rateLimiter,
	}, handlers)

	// Collect resources that own goroutines / connections so Run can tear them
	// down on shutdown. The workflow engine (Hatchet/Temporal worker) and cron
	// scheduler expose Close() via io.Closer; pubsub always does.
	// repos.close (pool/sql.DB teardown) is registered first so LIFO shutdown
	// closes the database last, after every worker that might still use it.
	closers := []io.Closer{closerFunc(func() error { repos.close(); return nil }), pubsub}
	// The jetstream usage consumer shares the NATS connection; register it AFTER
	// pubsub so LIFO shutdown stops the consume loop before the connection drains.
	if ingestCloser != nil {
		closers = append(closers, ingestCloser)
	}
	if c, ok := engine.(io.Closer); ok {
		closers = append(closers, c)
	}
	if c, ok := scheduler.(io.Closer); ok {
		closers = append(closers, c)
	}
	// The Redis-backed rate limiter owns a dedicated connection pool; close it
	// on shutdown. The in-memory limiter holds no resources and is not a Closer,
	// so this is a no-op for that backend.
	if c, ok := rateLimiter.(io.Closer); ok {
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
