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
			mcp.Description("Document type: invoice, proforma, quote, receipt, statement"),
		),
		mcp.WithString("invoice_type",
			mcp.Required(),
			mcp.Description("Invoice type: initial, recurring, usage, adjustment, setup, cancellation, refund"),
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

// NewGetInvoiceTool creates a tool to get an invoice by ID
func NewGetInvoiceTool() mcp.Tool {
	return mcp.NewTool("get_invoice",
		mcp.WithDescription("Get an invoice by ID"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice to retrieve"),
		),
	)
}

// NewListInvoicesTool creates a tool to list invoices
func NewListInvoicesTool() mcp.Tool {
	return mcp.NewTool("list_invoices",
		mcp.WithDescription("List invoices with pagination"),
		mcp.WithNumber("page",
			mcp.Description("Page number (default: 0)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of items per page (default: 10)"),
		),
		mcp.WithString("status",
			mcp.Description("Filter by invoice status (optional)"),
		),
		mcp.WithString("sort_by",
			mcp.Description("Field to sort by (default: created_at)"),
		),
		mcp.WithString("sort_order",
			mcp.Description("Sort order: asc or desc (default: desc)"),
		),
	)
}

// NewUpdateInvoiceTool creates a tool to update an invoice
func NewUpdateInvoiceTool() mcp.Tool {
	return mcp.NewTool("update_invoice",
		mcp.WithDescription("Update an existing invoice"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice to update"),
		),
		mcp.WithString("notes",
			mcp.Description("Notes for the invoice (optional)"),
		),
		mcp.WithString("customer_notes",
			mcp.Description("Notes for the customer (optional)"),
		),
		mcp.WithString("due_at",
			mcp.Description("Due date in ISO format (optional)"),
		),
		mcp.WithObject("metadata",
			mcp.Description("Additional metadata (optional)"),
		),
	)
}

// NewPerformInvoiceActionTool creates a tool to perform actions on an invoice
func NewPerformInvoiceActionTool() mcp.Tool {
	return mcp.NewTool("perform_invoice_action",
		mcp.WithDescription("Perform an action on an invoice (finalize, void, send, etc.)"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice"),
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action to perform: finalize, send, void, etc."),
		),
		mcp.WithString("reason",
			mcp.Description("Reason for the action (optional)"),
		),
	)
}

// NewListInvoiceLineItemsTool creates a tool to list invoice line items
func NewListInvoiceLineItemsTool() mcp.Tool {
	return mcp.NewTool("list_invoice_line_items",
		mcp.WithDescription("List line items for an invoice"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice"),
		),
	)
}

// NewAddInvoiceLineItemTool creates a tool to add a line item to an invoice
func NewAddInvoiceLineItemTool() mcp.Tool {
	return mcp.NewTool("add_invoice_line_item",
		mcp.WithDescription("Add a line item to an invoice"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice"),
		),
		mcp.WithString("product_id",
			mcp.Description("ID of the product (optional)"),
		),
		mcp.WithString("variant_id",
			mcp.Description("ID of the variant (optional)"),
		),
		mcp.WithString("price_id",
			mcp.Description("ID of the price (optional)"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Description of the line item"),
		),
		mcp.WithString("category",
			mcp.Description("Category of the line item (optional)"),
		),
		mcp.WithNumber("quantity",
			mcp.Required(),
			mcp.Description("Quantity of the line item"),
		),
		mcp.WithNumber("unit_price",
			mcp.Required(),
			mcp.Description("Unit price in cents"),
		),
		mcp.WithString("discount_type",
			mcp.Description("Discount type: percentage or fixed (optional)"),
		),
		mcp.WithNumber("discount_value",
			mcp.Description("Discount value (optional)"),
		),
		mcp.WithString("tax_code",
			mcp.Description("Tax code (optional)"),
		),
		mcp.WithNumber("tax_rate",
			mcp.Description("Tax rate in basis points (optional)"),
		),
		mcp.WithBoolean("tax_exempt",
			mcp.Description("Whether the item is tax exempt (optional)"),
		),
		mcp.WithObject("metadata",
			mcp.Description("Additional metadata (optional)"),
		),
	)
}

// NewUpdateInvoiceLineItemTool creates a tool to update an invoice line item
func NewUpdateInvoiceLineItemTool() mcp.Tool {
	return mcp.NewTool("update_invoice_line_item",
		mcp.WithDescription("Update an invoice line item"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice"),
		),
		mcp.WithString("line_item_id",
			mcp.Required(),
			mcp.Description("ID of the line item"),
		),
		mcp.WithString("description",
			mcp.Description("Description of the line item (optional)"),
		),
		mcp.WithString("category",
			mcp.Description("Category of the line item (optional)"),
		),
		mcp.WithNumber("quantity",
			mcp.Description("Quantity of the line item (optional)"),
		),
		mcp.WithNumber("unit_price",
			mcp.Description("Unit price in cents (optional)"),
		),
		mcp.WithString("discount_type",
			mcp.Description("Discount type: percentage or fixed (optional)"),
		),
		mcp.WithNumber("discount_value",
			mcp.Description("Discount value (optional)"),
		),
		mcp.WithString("tax_code",
			mcp.Description("Tax code (optional)"),
		),
		mcp.WithNumber("tax_rate",
			mcp.Description("Tax rate in basis points (optional)"),
		),
		mcp.WithBoolean("tax_exempt",
			mcp.Description("Whether the item is tax exempt (optional)"),
		),
		mcp.WithObject("metadata",
			mcp.Description("Additional metadata (optional)"),
		),
	)
}

// NewDeleteInvoiceLineItemTool creates a tool to delete an invoice line item
func NewDeleteInvoiceLineItemTool() mcp.Tool {
	return mcp.NewTool("delete_invoice_line_item",
		mcp.WithDescription("Delete an invoice line item"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice"),
		),
		mcp.WithString("line_item_id",
			mcp.Required(),
			mcp.Description("ID of the line item to delete"),
		),
	)
}

// NewListInvoiceHistoryTool creates a tool to list invoice history
func NewListInvoiceHistoryTool() mcp.Tool {
	return mcp.NewTool("list_invoice_history",
		mcp.WithDescription("List the history of changes for an invoice"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice"),
		),
	)
}

// NewGenerateInvoicePDFTool creates a tool to generate an invoice PDF
func NewGenerateInvoicePDFTool() mcp.Tool {
	return mcp.NewTool("generate_invoice_pdf",
		mcp.WithDescription("Generate a PDF for an invoice"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice"),
		),
		mcp.WithString("template_name",
			mcp.Required(),
			mcp.Description("Name of the PDF template to use"),
		),
		mcp.WithString("output_path",
			mcp.Description("Output path for the PDF file (optional)"),
		),
	)
}

// NewCreateInvoicePaymentLinkTool creates a tool to create a payment link for an invoice
func NewCreateInvoicePaymentLinkTool() mcp.Tool {
	return mcp.NewTool("create_invoice_payment_link",
		mcp.WithDescription("Create a single-use payment link for an invoice"),
		mcp.WithString("invoice_id",
			mcp.Required(),
			mcp.Description("ID of the invoice"),
		),
		mcp.WithString("expires_at",
			mcp.Description("Expiration timestamp in ISO format (optional, defaults to invoice due date)"),
		),
		mcp.WithString("success_url",
			mcp.Description("URL to redirect to after successful payment (optional)"),
		),
		mcp.WithString("cancel_url",
			mcp.Description("URL to redirect to after cancelled payment (optional)"),
		),
		mcp.WithObject("config",
			mcp.Description("Additional payment link configuration overrides (optional)"),
		),
	)
}

// NewListCustomerInvoicesTool creates a tool to list invoices for a specific customer
func NewListCustomerInvoicesTool() mcp.Tool {
	return mcp.NewTool("list_customer_invoices",
		mcp.WithDescription("List invoices for a specific customer"),
		mcp.WithString("customer_id",
			mcp.Required(),
			mcp.Description("ID of the customer"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number (default: 0)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of items per page (default: 10)"),
		),
		mcp.WithString("status",
			mcp.Description("Filter by invoice status (optional)"),
		),
		mcp.WithString("sort_by",
			mcp.Description("Field to sort by (default: created_at)"),
		),
		mcp.WithString("sort_order",
			mcp.Description("Sort order: asc or desc (default: desc)"),
		),
	)
}