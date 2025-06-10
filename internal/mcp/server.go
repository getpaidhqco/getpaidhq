package mcp

import (
	"context"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/mcp/handlers"
	"payloop/internal/mcp/tools"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServer represents the MCP server instance
type MCPServer struct {
	SSEServer      *server.SSEServer
	logger         logger.Logger
	invoiceService interfaces.InvoiceService
}

// NewServer creates a new MCP server with the provided dependencies
func NewServer(logger logger.Logger, invoiceService interfaces.InvoiceService) MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		"payloop-mcp 🚀",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Register hello world tool
	helloTool := tools.NewHelloWorldTool()
	s.AddTool(helloTool, handlers.HelloHandler)

	// Register create invoice tool
	createInvoiceTool := tools.NewCreateInvoiceTool()

	// Create a closure that captures the invoice service
	createInvoiceHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlers.InvoiceHandler(ctx, request, invoiceService, logger)
	}

	// Add create invoice tool handler
	s.AddTool(createInvoiceTool, createInvoiceHandler)

	// Start the SSE server
	sse := server.NewSSEServer(s,
		server.WithBaseURL(":8084"),
	)

	return MCPServer{
		SSEServer:      sse,
		logger:         logger,
		invoiceService: invoiceService,
	}
}
