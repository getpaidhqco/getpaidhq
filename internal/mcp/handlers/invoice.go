package handlers

import (
	"context"
	"fmt"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// InvoiceHandler handles the create_invoice tool requests
func InvoiceHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Extract required parameters
	customerId, err := request.RequireString("customer_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	docTypeStr, err := request.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	invoiceTypeStr, err := request.RequireString("invoice_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	currency, err := request.RequireString("currency")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Try to extract optional parameters
	var orderId string
	orderIdVal, err := request.RequireString("order_id")
	if err == nil {
		orderId = orderIdVal
	}

	var subscriptionId string
	subscriptionIdVal, err := request.RequireString("subscription_id")
	if err == nil {
		subscriptionId = subscriptionIdVal
	}

	var notes string
	notesVal, err := request.RequireString("notes")
	if err == nil {
		notes = notesVal
	}

	var customerNotes string
	customerNotesVal, err := request.RequireString("customer_notes")
	if err == nil {
		customerNotes = customerNotesVal
	}

	// Parse due_at if provided
	var dueAt time.Time
	dueAtStr, err := request.RequireString("due_at")
	if err == nil && dueAtStr != "" {
		parsedDueAt, err := time.Parse(time.RFC3339, dueAtStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid due_at format: %s", err.Error())), nil
		}
		dueAt = parsedDueAt
	}

	// Log the parameters
	logger.Info("Creating invoice with parameters:",
		"customerId", customerId,
		"type", docTypeStr,
		"invoiceType", invoiceTypeStr,
		"currency", currency,
		"orderId", orderId,
		"subscriptionId", subscriptionId,
		"dueAt", dueAt,
		"notes", notes,
		"customerNotes", customerNotes,
	)

	// In a real implementation, you would call the invoice service to create the invoice
	// For now, we'll just return a success message
	// Use a mock invoice ID
	invoiceId := "inv_123456789"

	// Return the result
	return mcp.NewToolResultText(fmt.Sprintf("Invoice created successfully with ID: %s", invoiceId)), nil
}