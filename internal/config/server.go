package config

import (
	"net/http"
	"time"

	"github.com/go-fuego/fuego"
	"github.com/go-playground/validator/v10"

	handler "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/adapter/http/middleware"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// Handlers groups every HTTP handler the application registers. The full
// app builds these against live services in NewApp. Route registration
// reads only metadata, so nil services are also safe for spec-only
// constructions (Fuego writes the OpenAPI spec to openapi.json at the
// repo root on s.Run() — see opts below).
type Handlers struct {
	Health         *handler.HealthHandler
	Order          *handler.OrderHandler
	Subscription   *handler.SubscriptionHandler
	Customer       *handler.CustomerHandler
	Product        *handler.ProductHandler
	Cart           *handler.CartHandler
	Session        *handler.SessionHandler
	Webhook        *handler.WebhookHandler
	WebhookSub     *handler.WebhookSubscriptionHandler
	Org            *handler.OrgHandler
	Psp            *handler.PspHandler
	PaymentMethod  *handler.PaymentMethodHandler
	Dunning        *handler.DunningHandler
	ApiKey         *handler.ApiKeyHandler
	ReminderConfig *handler.ReminderConfigHandler
	Usage          *handler.UsageHandler
	Meter          *handler.MeterHandler
	Invoice        *handler.InvoiceHandler
	Payment        *handler.PaymentHandler
	Setting        *handler.SettingHandler
}

// ServerDeps groups the cross-cutting wiring the server needs that is
// independent of business handlers: middleware, the validator instance,
// and the listen address.
type ServerDeps struct {
	Addr           string
	Logger         port.Logger
	Validator      *validator.Validate
	Authenticators []port.Authenticator
	Env            lib.Env
	// RateLimiter backs the per-client rate-limit middleware. When nil (e.g.
	// the OpenAPI exporter) the middleware is a pass-through regardless of
	// RATE_LIMIT_RPS.
	RateLimiter port.RateLimiter
}

// BuildServer wires every HTTP route onto a fresh *fuego.Server. Used by
// App.Run. Side-effecting middleware is attached here so the same
// configuration is reflected in the generated OpenAPI spec that Fuego
// writes on startup.
func BuildServer(deps ServerDeps, h Handlers) *fuego.Server {
	// Global middleware ordering. fuego.WithGlobalMiddlewares executes the LAST
	// entry OUTERMOST (first on the way in), so to make a middleware run
	// earlier at request time, append it LATER here.
	//
	// Request-time order:  CORS → rate-limit → authn → router
	//
	// Why CORS is OUTERMOST: rs/cors writes Access-Control-Allow-Origin on the
	// way in, before delegating to next. If CORS sits inside authn (or rate-
	// limit), a 401 / 429 emitted by an inner layer is returned WITHOUT CORS
	// headers — and the browser then surfaces it as "Failed to fetch" instead
	// of a debuggable HTTP error. The outermost CORS layer guarantees every
	// response, success or failure, carries the correct CORS headers.
	//
	// Why rate-limit sits inside CORS but outside authn: it sheds abusive
	// callers before they reach the (relatively expensive) authenticator
	// chain and protects the auth path itself from brute-force / floods.
	// Keyed by the securely-resolved client IP (same trusted-proxy rules the
	// rest of the app uses). Opt-in: a non-positive RATE_LIMIT_RPS (or nil
	// backend) leaves it a pass-through. ParseTrustedProxies already ran (and
	// validated) in NewApp; re-parsing here cannot see malformed input in
	// the live path, and the KeyFunc is never invoked while the limiter is
	// disabled.
	mws := []func(http.Handler) http.Handler{}

	// Authn (innermost of the three) is optional so the exporter can boot
	// without a Clerk key.
	if len(deps.Authenticators) > 0 {
		mws = append(mws,
			middleware.NewAuthnWrapperMiddleware(deps.Authenticators, deps.Logger, deps.Env).Handler(),
		)
	}

	trustedProxies, _ := handler.ParseTrustedProxies(deps.Env.TrustedProxies)
	rateLimiter := middleware.NewRateLimitMiddleware(deps.Logger, deps.RateLimiter, middleware.RateLimitConfig{
		RPS:     deps.Env.RateLimitRPS,
		Burst:   deps.Env.RateLimitBurst,
		KeyFunc: func(r *http.Request) string { return handler.ClientIP(r, trustedProxies) },
	})
	if rateLimiter.Enabled() {
		mws = append(mws, rateLimiter.Handler())
	}

	// CORS appended LAST so it runs OUTERMOST — see the ordering note above.
	mws = append(mws, middleware.NewCorsMiddleware(deps.Logger, deps.Env).Handler())

	opts := []fuego.ServerOption{
		fuego.WithErrorSerializer(handler.ApiErrorSerializer),
		fuego.WithGlobalMiddlewares(mws...),
		fuego.WithoutStartupMessages(),
		// Canonical OpenAPI artifact: ./openapi.json at the repo root.
		// Fuego writes it on s.Run() (engine.go:OutputOpenAPISpec). The default
		// would land it at doc/openapi.json; we keep it at the project root
		// because it is the API contract every other repo in the workspace
		// consumes (getpaidhq-sdk, gphq-web, gphq-checkout). The same in-
		// memory spec is also served at runtime from /swagger/openapi.json
		// for Swagger UI — that endpoint is not read from disk.
		fuego.WithEngineOptions(fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
			JSONFilePath: "openapi.json",
		})),
	}
	if deps.Validator != nil {
		opts = append(opts, fuego.WithValidator(deps.Validator))
	}
	if deps.Addr != "" {
		opts = append(opts, fuego.WithAddr(deps.Addr))
	}

	s := fuego.NewServer(opts...)

	// HTTP server timeouts. Without these, slowloris-style attacks
	// (slow header send, slow body send, slow body read) trivially
	// exhaust the goroutine table. Fuego embeds *http.Server directly,
	// so we set the fields after construction. Values match common
	// reverse-proxy defaults — tune to your traffic profile if you
	// upload large payloads.
	s.Server.ReadHeaderTimeout = 5 * time.Second
	s.Server.ReadTimeout = 30 * time.Second
	s.Server.WriteTimeout = 60 * time.Second
	s.Server.IdleTimeout = 120 * time.Second

	// No tag on the /api group itself — Fuego's spec validator rejects empty
	// identifiers, and every child group (e.g. /customers, /orders) sets its
	// own real tag for the dashboard.
	api := fuego.Group(s, "/api")
	registerAll(api, h)
	return s
}

func registerAll(api *fuego.Server, h Handlers) {
	if h.Health != nil {
		h.Health.RegisterRoutes(api)
	}
	if h.Order != nil {
		h.Order.RegisterRoutes(api)
	}
	if h.Subscription != nil {
		h.Subscription.RegisterRoutes(api)
	}
	if h.Customer != nil {
		h.Customer.RegisterRoutes(api)
	}
	if h.Product != nil {
		h.Product.RegisterRoutes(api)
	}
	if h.Cart != nil {
		h.Cart.RegisterRoutes(api)
	}
	if h.Session != nil {
		h.Session.RegisterRoutes(api)
	}
	if h.Webhook != nil {
		h.Webhook.RegisterRoutes(api)
	}
	if h.WebhookSub != nil {
		h.WebhookSub.RegisterRoutes(api)
	}
	if h.Org != nil {
		h.Org.RegisterRoutes(api)
	}
	if h.Psp != nil {
		h.Psp.RegisterRoutes(api)
	}
	if h.PaymentMethod != nil {
		h.PaymentMethod.RegisterRoutes(api)
	}
	if h.Dunning != nil {
		h.Dunning.RegisterRoutes(api)
	}
	if h.ApiKey != nil {
		h.ApiKey.RegisterRoutes(api)
	}
	if h.ReminderConfig != nil {
		h.ReminderConfig.RegisterRoutes(api)
	}
	if h.Usage != nil {
		h.Usage.RegisterRoutes(api)
	}
	if h.Meter != nil {
		h.Meter.RegisterRoutes(api)
	}
	if h.Invoice != nil {
		h.Invoice.RegisterRoutes(api)
	}
	if h.Payment != nil {
		h.Payment.RegisterRoutes(api)
	}
	if h.Setting != nil {
		h.Setting.RegisterRoutes(api)
	}
}
