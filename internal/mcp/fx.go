package mcp

import (
	"go.uber.org/fx"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
)

// NewServerParams defines the parameters for creating a new MCP server
type NewServerParams struct {
	fx.In

	Logger         logger.Logger
	InvoiceService interfaces.InvoiceService
}

// NewServerWithParams creates a new MCP server with the provided parameters
func NewServerWithParams(params NewServerParams) MCPServer {
	return NewServer(params.Logger, params.InvoiceService)
}

// Module exports dependency
var Module = fx.Options(
	fx.Provide(NewServerWithParams),
)
