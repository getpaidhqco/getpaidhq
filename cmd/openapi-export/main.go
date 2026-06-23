// Command openapi-export writes the OpenAPI contract to docs/openapi.yml.
//
// It boots only the HTTP route registrations with nil services. Route
// registration reads handler metadata (paths, types, summaries) but never
// invokes a service, so the spec emerges without any database, NATS, Redis,
// or workflow-engine connection. This is the single, intentional way to
// regenerate the committed spec — the running server never writes it to disk
// (see internal/config/server.go: DisableLocalSave) and serves the live spec
// from memory at GET /openapi.json.
package main

import (
	"fmt"
	"net/http"
	"os"

	"sigs.k8s.io/yaml"

	handler "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/config"
	"getpaidhq/internal/lib"
)

const specOutputPath = "docs/openapi.yml"

func main() {
	logger := lib.GetLogger()

	// Every handler is constructed with nil dependencies — RegisterRoutes reads
	// only function references and route metadata, never the underlying service,
	// so nil is safe.
	handlers := config.Handlers{
		Health:         handler.NewHealthHandler(logger),
		Order:          handler.NewOrderHandler(nil, logger, nil, func(h http.Handler) http.Handler { return h }),
		Subscription:   handler.NewSubscriptionHandler(nil, logger, nil),
		Customer:       handler.NewCustomerHandler(nil, logger, nil),
		Product:        handler.NewProductHandler(nil, logger, nil),
		Cart:           handler.NewCartHandler(nil, logger, nil),
		Session:        handler.NewSessionHandler(nil, logger, nil),
		Webhook:        handler.NewWebhookHandler(nil, logger),
		WebhookSub:     handler.NewWebhookSubscriptionHandler(nil, logger, nil),
		Org:            handler.NewOrgHandler(nil, logger),
		Psp:            handler.NewPspHandler(nil, logger, nil),
		PaymentMethod:  handler.NewPaymentMethodHandler(nil),
		Dunning:        handler.NewDunningHandler(nil, nil, logger, nil, nil),
		ApiKey:         handler.NewApiKeyHandler(nil, logger, nil),
		ReminderConfig: handler.NewReminderConfigHandler(nil, logger),
		Usage:          handler.NewUsageHandler(nil, logger, nil),
		Meter:          handler.NewMeterHandler(nil, logger, nil),
		Invoice:        handler.NewInvoiceHandler(nil, logger, nil),
		Payment:        handler.NewPaymentHandler(nil, logger, nil),
		Setting:        handler.NewSettingHandler(nil, logger, nil),
		Coupon:         handler.NewCouponHandler(nil, logger, nil),
	}

	server := config.BuildServer(config.ServerDeps{
		Logger:    logger,
		Validator: lib.NewValidator(),
	}, handlers)
	defer server.Server.Close()

	// OutputOpenAPISpec computes tags and validates the spec. DisableLocalSave
	// is set in BuildServer, so this does not write any file itself.
	spec := server.OutputOpenAPISpec()
	spec.Info.Title = "GetPaidHQ API"
	spec.Info.Version = "1.0.0"

	jsonBytes, err := spec.MarshalJSON()
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal openapi to json:", err)
		os.Exit(1)
	}
	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "convert json to yaml:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(specOutputPath, yamlBytes, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write "+specOutputPath+":", err)
		os.Exit(1)
	}
	fmt.Println("wrote", specOutputPath)
}
