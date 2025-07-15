package tools

import (
	"payloop/internal/application/dto"
	"payloop/internal/mcp/schema"

	"github.com/mark3labs/mcp-go/mcp"
)

// Order tool generator
var orderToolGenerator = schema.NewGenerator()

// NewCreateOrderTool creates a new order creation tool with schema from DTO
func NewCreateOrderTool() mcp.Tool {
	tool, err := orderToolGenerator.GenerateToolFromDTO(
		"create_order",
		"Create a new order for a customer. Organization ID is automatically extracted from authentication.",
		dto.CreateOrderInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("create_order",
			mcp.WithDescription("Create a new order for a customer"),
		)
	}
	return tool
}

// NewGetOrderTool creates an order retrieval tool
func NewGetOrderTool() mcp.Tool {
	return mcp.NewTool("get_order",
		mcp.WithDescription("Retrieve an order by ID"),
		mcp.WithString("order_id",
			mcp.Required(),
			mcp.Description("Order ID to retrieve"),
		),
	)
}

// NewListOrdersTool creates an order listing tool with schema
func NewListOrdersTool() mcp.Tool {
	tool, err := orderToolGenerator.GenerateToolFromDTO(
		"list_orders",
		"List orders with optional filtering and pagination",
		dto.OrderListFilters{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("list_orders",
			mcp.WithDescription("List orders with optional filtering and pagination"),
		)
	}
	return tool
}

// NewCompleteOrderTool creates an order completion tool with schema
func NewCompleteOrderTool() mcp.Tool {
	tool, err := orderToolGenerator.GenerateToolFromDTO(
		"complete_order",
		"Complete an order by processing payment and fulfillment",
		dto.CompleteOrderInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("complete_order",
			mcp.WithDescription("Complete an order by processing payment and fulfillment"),
		)
	}
	return tool
}

// NewListOrderSubscriptionsTool creates a tool to list subscriptions for an order
func NewListOrderSubscriptionsTool() mcp.Tool {
	return mcp.NewTool("list_order_subscriptions",
		mcp.WithDescription("List all subscriptions created from an order"),
		mcp.WithString("order_id",
			mcp.Required(),
			mcp.Description("Order ID to list subscriptions for"),
		),
	)
}