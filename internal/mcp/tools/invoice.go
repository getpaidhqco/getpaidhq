package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// NewCreateInvoiceTool creates a new create invoice tool
func NewCreateInvoiceTool() mcp.Tool {
	return mcp.NewTool("create_invoice",
		mcp.WithDescription("Create a new invoice"),
		mcp.WithString("customer_id",
			mcp.Required(),
			mcp.Description("ID of the customer"),
		),
		mcp.WithString("order_id",
			mcp.Description("ID of the order (optional)"),
		),
		mcp.WithString("subscription_id",
			mcp.Description("ID of the subscription (optional)"),
		),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Document type (e.g., INVOICE)"),
		),
		mcp.WithString("invoice_type",
			mcp.Required(),
			mcp.Description("Invoice type (e.g., STANDARD)"),
		),
		mcp.WithString("currency",
			mcp.Required(),
			mcp.Description("Currency code (e.g., USD)"),
		),
		mcp.WithString("due_at",
			mcp.Description("Due date in ISO format (optional)"),
		),
		mcp.WithString("notes",
			mcp.Description("Notes for the invoice (optional)"),
		),
		mcp.WithString("customer_notes",
			mcp.Description("Notes for the customer (optional)"),
		),
		mcp.WithObject("metadata",
			mcp.Description("Additional metadata (optional)"),
		),
		mcp.WithArray("line_items",
			mcp.Description("Line items for the invoice (optional)"),
		),
	)
}