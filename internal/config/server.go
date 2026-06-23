package config

import (
	"net/http"
	"time"

	"github.com/go-fuego/fuego"
	"github.com/go-playground/validator/v10"

	handler "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/adapter/http/middleware"
	"getpaidhq/internal/core/port"
)

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
	Coupon         *handler.CouponHandler
}

type ServerDeps struct {
	Addr           string
	Logger         port.Logger
	Validator      *validator.Validate
	Authenticators []port.Authenticator
	// AllowedOrigins is the raw comma-separated CORS origin list (ALLOWED_ORIGINS).
	AllowedOrigins string
	// TrustedProxies is the raw comma-separated CIDR list (TRUSTED_PROXIES)
	// used to key the rate limiter by securely-resolved client IP.
	TrustedProxies string
	// RateLimitRPS / RateLimitBurst configure the per-client rate limit
	// (RATE_LIMIT_RPS / RATE_LIMIT_BURST). RPS <= 0 disables it.
	RateLimitRPS   int
	RateLimitBurst int
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
			middleware.NewAuthnWrapperMiddleware(deps.Authenticators, deps.Logger).Handler(),
		)
	}

	trustedProxies, _ := handler.ParseTrustedProxies(deps.TrustedProxies)
	rateLimiter := middleware.NewRateLimitMiddleware(deps.Logger, deps.RateLimiter, middleware.RateLimitConfig{
		RPS:     deps.RateLimitRPS,
		Burst:   deps.RateLimitBurst,
		KeyFunc: func(r *http.Request) string { return handler.ClientIP(r, trustedProxies) },
	})
	if rateLimiter.Enabled() {
		mws = append(mws, rateLimiter.Handler())
	}

	// CORS appended LAST so it runs OUTERMOST — see the ordering note above.
	mws = append(mws, middleware.NewCorsMiddleware(deps.Logger, deps.AllowedOrigins).Handler())

	opts := []fuego.ServerOption{
		fuego.WithErrorSerializer(handler.ApiErrorSerializer),
		// Pass our own ApiError envelope through untouched. Fuego's default
		// engine ErrorHandler coerces any error implementing ErrorWithStatus
		// (ApiError does) into a fuego.HTTPError BEFORE the serializer runs,
		// discarding our Message/Details and filling Title from http.StatusText
		// — so every custom error degraded to {"message":"Bad Request","details":null}.
		// Everything that is NOT our ApiError still goes through Fuego's default
		// handler, preserving its status normalization for fuego.* error types
		// and the passthrough for plain errors.
		fuego.WithEngineOptions(fuego.WithErrorHandler(handler.PassThroughApiError)),
		fuego.WithGlobalMiddlewares(mws...),
		fuego.WithoutStartupMessages(),
		// OpenAPI: the running server serves the live, in-memory spec as JSON
		// at GET /openapi.json and nothing else — no Swagger UI, and no file is
		// ever written to disk (DisableLocalSave). Booting the server therefore
		// produces zero git churn. The committed contract is generated on demand
		// by `make openapi` (cmd/openapi-export -> docs/openapi.yml).
		fuego.WithEngineOptions(fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
			SpecURL:          "/openapi.json",
			DisableLocalSave: true,
			DisableSwaggerUI: true,
		})),
		// Render `validate:"oneof=..."` fields as real OpenAPI enums (not opaque
		// strings) so the spec is constrained and SDK clients get enum types.
		fuego.WithEngineOptions(fuego.WithOpenAPIGeneratorSchemaCustomizer(handler.EnumSchemaCustomizer)),
		// Declare request bodies as application/json. Without this Fuego defaults
		// the consumed content type to */*, which strict OpenAPI client generators
		// (e.g. ogen) reject — they skip every operation with a */* body, silently
		// dropping coverage. Pinning it to application/json keeps generated clients
		// (CLI, SDK) at full API coverage.
		fuego.WithEngineOptions(fuego.WithRequestContentType("application/json")),
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
	if h.Coupon != nil {
		h.Coupon.RegisterRoutes(api)
	}
}
