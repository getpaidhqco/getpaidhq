package mcp

import (
	"context"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"payloop/internal/mcp/handlers"
	"payloop/internal/mcp/middleware"
	"payloop/internal/mcp/tools"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServer represents the MCP server instance
type MCPServer struct {
	SSEServer                        *server.SSEServer
	logger                           logger.Logger
	authService                      *middleware.AuthService
	invoiceService                   interfaces.InvoiceService
	customerService                  interfaces.CustomerService
	subscriptionOrchestrationService interfaces.SubscriptionOrchestrationService
	orderService                     interfaces.OrderService
	productService                   interfaces.ProductService
}

// NewServer creates a new MCP server with the provided dependencies
func NewServer(
	logger logger.Logger,
	env lib.Env,
	apiKeyRepository repositories.ApiKeyRepository,
	metadataRepository repositories.MetadataStoreRepository,
	invoiceService interfaces.InvoiceService,
	customerService interfaces.CustomerService,
	subscriptionOrchestrationService interfaces.SubscriptionOrchestrationService,
	orderService interfaces.OrderService,
	productService interfaces.ProductService,
) MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		"payloop-mcp 🚀",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Create authentication service
	authService := middleware.NewAuthService(
		logger,
		env,
		apiKeyRepository,
		metadataRepository,
	)

	// === CUSTOMER TOOLS ===
	s.AddTool(tools.NewCreateCustomerTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.CreateCustomerHandler(ctx, request, customerService, authService, logger)
	})
	s.AddTool(tools.NewGetCustomerTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.GetCustomerHandler(ctx, request, customerService, authService, logger)
	})
	s.AddTool(tools.NewListCustomersTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.ListCustomersHandler(ctx, request, customerService, authService, logger)
	})
	s.AddTool(tools.NewUpdateCustomerTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.UpdateCustomerHandler(ctx, request, customerService, authService, logger)
	})
	s.AddTool(tools.NewCreatePaymentMethodTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.CreatePaymentMethodHandler(ctx, request, customerService, authService, logger)
	})
	s.AddTool(tools.NewUpdatePaymentMethodTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.UpdatePaymentMethodHandler(ctx, request, customerService, authService, logger)
	})
	s.AddTool(tools.NewGetPaymentMethodTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.GetPaymentMethodHandler(ctx, request, customerService, authService, logger)
	})

	// === PRODUCT TOOLS ===
	s.AddTool(tools.NewCreateProductTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.CreateProductHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewGetProductTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.GetProductHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewListProductsTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.ListProductsHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewUpdateProductTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.UpdateProductHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewDeleteProductTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.DeleteProductHandler(ctx, request, productService, authService, logger)
	})

	// === VARIANT TOOLS ===
	s.AddTool(tools.NewCreateVariantTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.CreateVariantHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewGetVariantTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.GetVariantHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewListVariantsTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.ListVariantsHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewUpdateVariantTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.UpdateVariantHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewDeleteVariantTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.DeleteVariantHandler(ctx, request, productService, authService, logger)
	})

	// === PRICE TOOLS ===
	s.AddTool(tools.NewCreatePriceTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.CreatePriceHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewGetPriceTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.GetPriceHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewListPricesTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.ListPricesHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewUpdatePriceTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.UpdatePriceHandler(ctx, request, productService, authService, logger)
	})
	s.AddTool(tools.NewDeletePriceTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.DeletePriceHandler(ctx, request, productService, authService, logger)
	})

	// === ORDER TOOLS ===
	s.AddTool(tools.NewCreateOrderTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.CreateOrderHandler(ctx, request, orderService, authService, logger)
	})
	s.AddTool(tools.NewGetOrderTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.GetOrderHandler(ctx, request, orderService, authService, logger)
	})
	s.AddTool(tools.NewListOrdersTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.ListOrdersHandler(ctx, request, orderService, authService, logger)
	})
	s.AddTool(tools.NewCompleteOrderTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.CompleteOrderHandler(ctx, request, orderService, authService, logger)
	})
	s.AddTool(tools.NewListOrderSubscriptionsTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.ListOrderSubscriptionsHandler(ctx, request, orderService, authService, logger)
	})

	// === SUBSCRIPTION TOOLS ===
	s.AddTool(tools.NewCreateSubscriptionTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.CreateSubscriptionHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})
	s.AddTool(tools.NewGetSubscriptionTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.GetSubscriptionHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})
	s.AddTool(tools.NewListSubscriptionsTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.ListSubscriptionsHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})
	s.AddTool(tools.NewUpdateSubscriptionTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.UpdateSubscriptionHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})
	s.AddTool(tools.NewPauseSubscriptionTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.PauseSubscriptionHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})
	s.AddTool(tools.NewResumeSubscriptionTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.ResumeSubscriptionHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})
	s.AddTool(tools.NewCancelSubscriptionTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.CancelSubscriptionHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})
	s.AddTool(tools.NewChangePlanTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.ChangePlanHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})
	s.AddTool(tools.NewGetSubscriptionPaymentsTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.GetSubscriptionPaymentsHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})
	s.AddTool(tools.NewGetSubscriptionUsageTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.GetSubscriptionUsageHandler(ctx, request, subscriptionOrchestrationService, authService, logger)
	})

	// === INVOICE TOOLS ===
	s.AddTool(tools.NewCreateInvoiceTool(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.InvoiceHandler(ctx, request, invoiceService, authService, logger)
	})

	// Start the SSE server
	sse := server.NewSSEServer(s,
		server.WithBaseURL(":"+env.McpSsePort),
	)

	return MCPServer{
		SSEServer:                        sse,
		logger:                           logger,
		authService:                      authService,
		invoiceService:                   invoiceService,
		customerService:                  customerService,
		subscriptionOrchestrationService: subscriptionOrchestrationService,
		orderService:                     orderService,
		productService:                   productService,
	}
}