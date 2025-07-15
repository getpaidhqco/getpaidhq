package mcp

import (
	"go.uber.org/fx"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"payloop/internal/mcp/adapters"
)

// NewServerParams defines the parameters for creating a new MCP server
type NewServerParams struct {
	fx.In

	Logger                              logger.Logger
	Env                                 lib.Env
	ApiKeyRepository                    repositories.ApiKeyRepository
	MetadataRepository                  repositories.MetadataStoreRepository
	InvoiceService                      interfaces.InvoiceService
	CustomerService                     interfaces.CustomerService
	SubscriptionOrchestrationService    interfaces.SubscriptionOrchestrationService
	OrderService                        interfaces.OrderService
	ConcreteProductService              services.ProductService
}

// NewServerWithParams creates a new MCP server with the provided parameters
func NewServerWithParams(params NewServerParams) MCPServer {
	// Create adapter for product service
	productService := adapters.NewProductServiceAdapter(params.ConcreteProductService)
	
	return NewServer(
		params.Logger,
		params.Env,
		params.ApiKeyRepository,
		params.MetadataRepository,
		params.InvoiceService,
		params.CustomerService,
		params.SubscriptionOrchestrationService,
		params.OrderService,
		productService,
	)
}

// Module exports dependency
var Module = fx.Options(
	fx.Provide(NewServerWithParams),
)
