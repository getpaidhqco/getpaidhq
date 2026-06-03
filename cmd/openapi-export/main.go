// openapi-export writes the OpenAPI spec to openapi.yml at the repo
// root by booting the HTTP route registrations with nil services. Route
// registration only reads metadata, so the spec emerges without any
// database, NATS, or Redis connection.
package main

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"

	handler "getpaidhq/internal/adapter/http"
	"getpaidhq/internal/config"
	"getpaidhq/internal/lib"
)

func main() {
	logger := lib.GetLogger()

	// Every handler is constructed with nil dependencies — RegisterRoutes
	// reads only function references and route metadata, never invokes
	// the underlying service, so nil is safe here.
	customer := handler.NewCustomerHandler(nil, logger, nil)
	handlers := config.Handlers{
		Health:        handler.NewHealthHandler(logger),
		Order:         handler.NewOrderHandler(nil, logger, nil),
		Subscription:  handler.NewSubscriptionHandler(nil, logger, nil),
		Customer:      customer,
		Product:       handler.NewProductHandler(nil, logger, nil),
		Cart:          handler.NewCartHandler(nil, logger, nil),
		Session:       handler.NewSessionHandler(nil, logger, nil),
		Webhook:       handler.NewWebhookHandler(nil, logger),
		WebhookSub:    handler.NewWebhookSubscriptionHandler(nil, logger, nil),
		Org:           handler.NewOrgHandler(nil, logger),
		Psp:           handler.NewPspHandler(nil, logger, nil),
		PaymentMethod: handler.NewPaymentMethodHandler(customer),
		Dunning:       handler.NewDunningHandler(nil, nil, logger, nil, nil),
	}

	server := config.BuildServer(config.ServerDeps{
		Logger:    logger,
		Validator: lib.NewValidator(),
	}, handlers)

	spec := server.OpenAPI.Description()
	spec.Info.Title = "Payloop API"
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
	if err := os.WriteFile("openapi.yml", yamlBytes, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write openapi.yml:", err)
		os.Exit(1)
	}
	fmt.Println("wrote openapi.yml")
}
