package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/lib/pdf"
	"payloop/internal/domain/entities"
	"payloop/internal/mcp/middleware"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// CreateInvoiceHandler handles the create_invoice tool requests
func CreateInvoiceHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

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

	// Parse document type and invoice type
	docType := entities.DocumentType(docTypeStr)
	invoiceType := entities.InvoiceType(invoiceTypeStr)

	// Extract optional parameters
	var orderId, subscriptionId, notes, customerNotes string
	if val, _ := request.RequireString("order_id"); val != "" {
		orderId = val
	}
	if val, _ := request.RequireString("subscription_id"); val != "" {
		subscriptionId = val
	}
	if val, _ := request.RequireString("notes"); val != "" {
		notes = val
	}
	if val, _ := request.RequireString("customer_notes"); val != "" {
		customerNotes = val
	}

	// Parse due_at if provided
	var dueAt time.Time
	if dueAtStr, _ := request.RequireString("due_at"); dueAtStr != "" {
		if parsedDueAt, err := time.Parse(time.RFC3339, dueAtStr); err == nil {
			dueAt = parsedDueAt
		}
	}

	// Build create input
	input := dto.CreateInvoiceInput{
		CustomerId:     customerId,
		OrderId:        orderId,
		SubscriptionId: subscriptionId,
		Type:           docType,
		InvoiceType:    invoiceType,
		Currency:       currency,
		DueAt:          dueAt,
		Notes:          notes,
		CustomerNotes:  customerNotes,
		Metadata:       make(map[string]string),
	}

	// Create the invoice
	invoice, err := invoiceService.Create(ctx, authCtx.OrgId, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create invoice: %s", err.Error())), nil
	}

	// Convert to JSON response
	response, err := json.Marshal(invoice)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}

// GetInvoiceHandler handles the get_invoice tool requests
func GetInvoiceHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the invoice
	invoice, err := invoiceService.Get(ctx, authCtx.OrgId, invoiceId)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get invoice: %s", err.Error())), nil
	}

	// Convert to JSON response
	response, err := json.Marshal(invoice)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}

// ListInvoicesHandler handles the list_invoices tool requests
func ListInvoicesHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract pagination parameters
	page := 0
	if pageStr, _ := request.RequireString("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	limit := 10
	if limitStr, _ := request.RequireString("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	sortBy := "created_at"
	if sortByVal, _ := request.RequireString("sort_by"); sortByVal != "" {
		sortBy = sortByVal
	}

	sortOrder := "desc"
	if sortOrderVal, _ := request.RequireString("sort_order"); sortOrderVal != "" {
		sortOrder = sortOrderVal
	}

	// Create pagination DTO
	pagination := dto.Pagination{
		Page:          page,
		Limit:         limit,
		Offset:        page * limit,
		SortBy:        sortBy,
		SortDirection: sortOrder,
	}

	// List invoices
	invoices, total, err := invoiceService.List(ctx, authCtx.OrgId, pagination)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list invoices: %s", err.Error())), nil
	}

	// Build response with pagination metadata
	response := map[string]interface{}{
		"data": invoices,
		"meta": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	// Convert to JSON response
	respData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(respData)), nil
}

// UpdateInvoiceHandler handles the update_invoice tool requests
func UpdateInvoiceHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Extract optional parameters
	input := dto.UpdateInvoiceRequest{}

	if notes, _ := request.RequireString("notes"); notes != "" {
		input.Notes = notes
	}
	if customerNotes, _ := request.RequireString("customer_notes"); customerNotes != "" {
		input.CustomerNotes = customerNotes
	}
	if dueAtStr, _ := request.RequireString("due_at"); dueAtStr != "" {
		if parsedDueAt, err := time.Parse(time.RFC3339, dueAtStr); err == nil {
			input.DueAt = parsedDueAt
		}
	}

	// Update the invoice
	invoice, err := invoiceService.Update(ctx, authCtx.OrgId, invoiceId, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update invoice: %s", err.Error())), nil
	}

	// Convert to JSON response
	response, err := json.Marshal(invoice)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}

// PerformInvoiceActionHandler handles the perform_invoice_action tool requests
func PerformInvoiceActionHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	action, err := request.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Extract optional parameters
	var reason string
	if reasonVal, _ := request.RequireString("reason"); reasonVal != "" {
		reason = reasonVal
	}

	// Build action input
	input := dto.InvoiceActionRequest{
		Action: action,
		Reason: reason,
	}

	// Perform the action
	invoice, err := invoiceService.PerformAction(ctx, authCtx.OrgId, invoiceId, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to perform action: %s", err.Error())), nil
	}

	// Convert to JSON response
	response, err := json.Marshal(invoice)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}

// ListInvoiceLineItemsHandler handles the list_invoice_line_items tool requests
func ListInvoiceLineItemsHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// List line items
	lineItems, err := invoiceService.ListLineItems(ctx, authCtx.OrgId, invoiceId)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list line items: %s", err.Error())), nil
	}

	// Convert to JSON response
	response, err := json.Marshal(lineItems)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}

// AddInvoiceLineItemHandler handles the add_invoice_line_item tool requests
func AddInvoiceLineItemHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	description, err := request.RequireString("description")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	quantityStr, err := request.RequireString("quantity")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	quantity, err := strconv.ParseFloat(quantityStr, 64)
	if err != nil {
		return mcp.NewToolResultError("Invalid quantity format"), nil
	}

	unitPriceStr, err := request.RequireString("unit_price")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	unitPrice, err := strconv.ParseFloat(unitPriceStr, 64)
	if err != nil {
		return mcp.NewToolResultError("Invalid unit_price format"), nil
	}

	// Build line item input
	input := dto.CreateInvoiceLineItemInput{
		Description: description,
		Quantity:    quantity,
		UnitPrice:   int(unitPrice),
	}

	// Extract optional parameters
	if val, _ := request.RequireString("product_id"); val != "" {
		input.ProductId = val
	}
	if val, _ := request.RequireString("variant_id"); val != "" {
		input.VariantId = val
	}
	if val, _ := request.RequireString("price_id"); val != "" {
		input.PriceId = val
	}
	if val, _ := request.RequireString("category"); val != "" {
		input.Category = val
	}

	// Add the line item
	lineItem, err := invoiceService.AddLineItem(ctx, authCtx.OrgId, invoiceId, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add line item: %s", err.Error())), nil
	}

	// Convert to JSON response
	response, err := json.Marshal(lineItem)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}

// UpdateInvoiceLineItemHandler handles the update_invoice_line_item tool requests
func UpdateInvoiceLineItemHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	lineItemId, err := request.RequireString("line_item_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build update input with optional parameters
	input := dto.UpdateInvoiceLineItemRequest{}

	if val, _ := request.RequireString("description"); val != "" {
		input.Description = val
	}
	if val, _ := request.RequireString("category"); val != "" {
		input.Category = val
	}
	if quantityStr, _ := request.RequireString("quantity"); quantityStr != "" {
		if quantity, err := strconv.ParseFloat(quantityStr, 64); err == nil {
			input.Quantity = quantity
		}
	}
	if unitPriceStr, _ := request.RequireString("unit_price"); unitPriceStr != "" {
		if unitPrice, err := strconv.ParseFloat(unitPriceStr, 64); err == nil {
			input.UnitPrice = int(unitPrice)
		}
	}

	// Update the line item
	lineItem, err := invoiceService.UpdateLineItem(ctx, authCtx.OrgId, invoiceId, lineItemId, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update line item: %s", err.Error())), nil
	}

	// Convert to JSON response
	response, err := json.Marshal(lineItem)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}

// DeleteInvoiceLineItemHandler handles the delete_invoice_line_item tool requests
func DeleteInvoiceLineItemHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	lineItemId, err := request.RequireString("line_item_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Delete the line item
	err = invoiceService.DeleteLineItem(ctx, authCtx.OrgId, invoiceId, lineItemId)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete line item: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(`{"status": "success"}`), nil
}

// ListInvoiceHistoryHandler handles the list_invoice_history tool requests
func ListInvoiceHistoryHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// List history
	history, err := invoiceService.ListHistory(ctx, authCtx.OrgId, invoiceId)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list history: %s", err.Error())), nil
	}

	// Convert to JSON response
	response, err := json.Marshal(history)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}

// GenerateInvoicePDFHandler handles the generate_invoice_pdf tool requests
func GenerateInvoicePDFHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	templateName, err := request.RequireString("template_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Extract optional parameters
	var outputPath string
	if val, _ := request.RequireString("output_path"); val != "" {
		outputPath = val
	}

	// Build PDF options
	options := pdf.GenerateOptions{
		TemplateName: templateName,
		OutputPath:   outputPath,
	}

	// Generate PDF
	pdfBytes, err := invoiceService.GeneratePDF(ctx, authCtx.OrgId, invoiceId, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate PDF: %s", err.Error())), nil
	}

	// Return success message with PDF size
	response := map[string]interface{}{
		"pdf_size": len(pdfBytes),
		"message":  fmt.Sprintf("PDF generated successfully for invoice %s", invoiceId),
	}

	respData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(respData)), nil
}

// CreateInvoicePaymentLinkHandler handles the create_invoice_payment_link tool requests
func CreateInvoicePaymentLinkHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	invoiceId, err := request.RequireString("invoice_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build payment link input
	input := dto.CreateInvoicePaymentLinkInput{}

	// Extract optional parameters
	if val, _ := request.RequireString("expires_at"); val != "" {
		if parsedTime, err := time.Parse(time.RFC3339, val); err == nil {
			input.ExpiresAt = parsedTime
		}
	}
	if val, _ := request.RequireString("success_url"); val != "" {
		input.SuccessUrl = val
	}
	if val, _ := request.RequireString("cancel_url"); val != "" {
		input.CancelUrl = val
	}

	// Create payment link
	paymentLink, err := invoiceService.CreatePaymentLink(ctx, authCtx.OrgId, invoiceId, input)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create payment link: %s", err.Error())), nil
	}

	// Convert to JSON response
	response, err := json.Marshal(paymentLink)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}

// ListCustomerInvoicesHandler handles the list_customer_invoices tool requests
func ListCustomerInvoicesHandler(ctx context.Context, request mcp.CallToolRequest, invoiceService interfaces.InvoiceService, authService *middleware.AuthService, logger logger.Logger) (*mcp.CallToolResult, error) {
	// Authenticate the request
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", err.Error())), nil
	}

	// Extract required parameters
	customerId, err := request.RequireString("customer_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Extract pagination parameters
	page := 0
	if pageStr, _ := request.RequireString("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	limit := 10
	if limitStr, _ := request.RequireString("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	sortBy := "created_at"
	if val, _ := request.RequireString("sort_by"); val != "" {
		sortBy = val
	}

	sortOrder := "desc"
	if val, _ := request.RequireString("sort_order"); val != "" {
		sortOrder = val
	}

	// Create pagination DTO
	pagination := dto.Pagination{
		Page:          page,
		Limit:         limit,
		Offset:        page * limit,
		SortBy:        sortBy,
		SortDirection: sortOrder,
	}

	// List customer invoices
	invoices, total, err := invoiceService.FindByCustomerId(ctx, authCtx.OrgId, customerId, pagination)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list customer invoices: %s", err.Error())), nil
	}

	// Build response with pagination metadata
	response := map[string]interface{}{
		"data": invoices,
		"meta": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	// Convert to JSON response
	respData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to serialize response: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(respData)), nil
}