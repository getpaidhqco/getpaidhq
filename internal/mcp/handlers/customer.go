package handlers

import (
	"context"
	"fmt"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/mcp/middleware"
	"payloop/internal/mcp/schema"

	"github.com/mark3labs/mcp-go/mcp"
)

var customerSchemaGenerator = schema.NewGenerator()

// CreateCustomerHandler handles customer creation requests with authentication
func CreateCustomerHandler(ctx context.Context, request mcp.CallToolRequest, 
                          customerService interfaces.CustomerService,
                          authService *middleware.AuthService,
                          logger logger.Logger) (*mcp.CallToolResult, error) {
    
    // Extract authentication from request arguments
    authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
    if err != nil {
        logger.Warn("Customer creation failed - authentication error", "error", err.Error())
        return mcp.NewToolResultError("Authentication required"), nil
    }

    // Parse arguments to DTO using MCP's built-in binding
    var input dto.CreateCustomerInput
    if err := request.BindArguments(&input); err != nil {
        logger.Error("Failed to parse customer creation input", 
            "error", err.Error(),
            "orgId", authCtx.OrgId)
        return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
    }

    // Log authenticated operation
    logger.Info("Creating customer via MCP",
        "orgId", authCtx.OrgId,
        "userId", authCtx.User.Id,
        "email", input.Email)

    // Call application service with authenticated org_id
    customer, err := customerService.Create(ctx, authCtx.OrgId, input)
    if err != nil {
        logger.Error("Failed to create customer",
            "orgId", authCtx.OrgId,
            "userId", authCtx.User.Id,
            "email", input.Email,
            "error", err.Error())
        return mcp.NewToolResultError(fmt.Sprintf("Failed to create customer: %s", err.Error())), nil
    }

    // Return success result
    return mcp.NewToolResultText(fmt.Sprintf("Customer created successfully. ID: %s, Email: %s", 
        customer.Id, customer.Email)), nil
}

// GetCustomerHandler handles customer retrieval requests
func GetCustomerHandler(ctx context.Context, request mcp.CallToolRequest,
                       customerService interfaces.CustomerService,
                       authService *middleware.AuthService,
                       logger logger.Logger) (*mcp.CallToolResult, error) {

    // Extract authentication
    authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
    if err != nil {
        return mcp.NewToolResultError("Authentication required"), nil
    }

    // Extract customer ID parameter using MCP helper
    customerId, err := request.RequireString("customer_id")
    if err != nil {
        return mcp.NewToolResultError("customer_id is required"), nil
    }

    // Log operation
    logger.Info("Retrieving customer via MCP",
        "orgId", authCtx.OrgId,
        "userId", authCtx.User.Id,
        "customerId", customerId)

    // Call service
    customer, err := customerService.Get(ctx, authCtx.OrgId, customerId)
    if err != nil {
        logger.Error("Failed to retrieve customer",
            "orgId", authCtx.OrgId,
            "customerId", customerId,
            "error", err.Error())
        return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve customer: %s", err.Error())), nil
    }

    // Return customer data as formatted text
    return mcp.NewToolResultText(fmt.Sprintf("Customer found - ID: %s, Email: %s, Name: %s %s, Created: %s",
        customer.Id, customer.Email, customer.FirstName, customer.LastName, customer.CreatedAt.Format("2006-01-02"))), nil
}

// ListCustomersHandler handles customer listing with filters
func ListCustomersHandler(ctx context.Context, request mcp.CallToolRequest,
                         customerService interfaces.CustomerService,
                         authService *middleware.AuthService,
                         logger logger.Logger) (*mcp.CallToolResult, error) {

    // Extract authentication
    authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
    if err != nil {
        return mcp.NewToolResultError("Authentication required"), nil
    }

    // Parse filter arguments
    type CustomerListInput struct {
        Page        int    `json:"page"`
        Limit       int    `json:"limit"`
        Status      string `json:"status"`
        EmailFilter string `json:"email_filter"`
    }

    var filters CustomerListInput
    if err := request.BindArguments(&filters); err != nil {
        logger.Error("Failed to parse customer list filters", "error", err.Error())
        return mcp.NewToolResultError(fmt.Sprintf("Invalid filters: %s", err.Error())), nil
    }

    // Set defaults
    if filters.Page <= 0 {
        filters.Page = 1
    }
    if filters.Limit <= 0 {
        filters.Limit = 20
    }
    if filters.Limit > 100 {
        filters.Limit = 100
    }

    // Create pagination DTO
    pagination := dto.Pagination{
        Page:   filters.Page,
        Limit:  filters.Limit,
        Offset: (filters.Page - 1) * filters.Limit,
    }

    // Log operation
    logger.Info("Listing customers via MCP",
        "orgId", authCtx.OrgId,
        "userId", authCtx.User.Id,
        "page", filters.Page,
        "limit", filters.Limit,
        "status", filters.Status,
        "emailFilter", filters.EmailFilter)

    // Call service
    result, err := customerService.List(ctx, authCtx.OrgId, pagination)
    if err != nil {
        logger.Error("Failed to list customers",
            "orgId", authCtx.OrgId,
            "error", err.Error())
        return mcp.NewToolResultError(fmt.Sprintf("Failed to list customers: %s", err.Error())), nil
    }

    // Format response
    if len(result.Items) == 0 {
        return mcp.NewToolResultText("No customers found"), nil
    }

    responseText := fmt.Sprintf("Found %d customers (total: %d, page: %d, limit: %d):\n", 
        len(result.Items), result.TotalCount, filters.Page, filters.Limit)
    
    for i, customer := range result.Items {
        responseText += fmt.Sprintf("%d. ID: %s, Email: %s, Name: %s %s\n",
            i+1, customer.Id, customer.Email, customer.FirstName, customer.LastName)
    }

    return mcp.NewToolResultText(responseText), nil
}

// UpdateCustomerHandler handles customer update requests
func UpdateCustomerHandler(ctx context.Context, request mcp.CallToolRequest,
                          customerService interfaces.CustomerService,
                          authService *middleware.AuthService,
                          logger logger.Logger) (*mcp.CallToolResult, error) {

    // Extract authentication
    authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
    if err != nil {
        return mcp.NewToolResultError("Authentication required"), nil
    }

    // Parse input with customer ID + update data
    type UpdateCustomerInput struct {
        CustomerID string `json:"customer_id"`
        dto.UpdateCustomerInput
    }

    var input UpdateCustomerInput
    if err := request.BindArguments(&input); err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
    }

    if input.CustomerID == "" {
        return mcp.NewToolResultError("customer_id is required"), nil
    }

    // Log operation
    logger.Info("Updating customer via MCP",
        "orgId", authCtx.OrgId,
        "userId", authCtx.User.Id,
        "customerId", input.CustomerID)

    // Call service
    customer, err := customerService.Update(ctx, authCtx.OrgId, input.CustomerID, input.UpdateCustomerInput)
    if err != nil {
        logger.Error("Failed to update customer",
            "orgId", authCtx.OrgId,
            "customerId", input.CustomerID,
            "error", err.Error())
        return mcp.NewToolResultError(fmt.Sprintf("Failed to update customer: %s", err.Error())), nil
    }

    return mcp.NewToolResultText(fmt.Sprintf("Customer updated successfully. ID: %s, Email: %s", 
        customer.Id, customer.Email)), nil
}

// CreatePaymentMethodHandler handles payment method creation
func CreatePaymentMethodHandler(ctx context.Context, request mcp.CallToolRequest,
                               customerService interfaces.CustomerService,
                               authService *middleware.AuthService,
                               logger logger.Logger) (*mcp.CallToolResult, error) {

    // Extract authentication
    authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
    if err != nil {
        return mcp.NewToolResultError("Authentication required"), nil
    }

    // Parse input
    var input dto.CreatePaymentMethodInput
    if err := request.BindArguments(&input); err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
    }

    // Log operation
    logger.Info("Creating payment method via MCP",
        "orgId", authCtx.OrgId,
        "userId", authCtx.User.Id,
        "customerId", input.CustomerId)

    // Call service
    paymentMethod, err := customerService.CreatePaymentMethod(ctx, authCtx.OrgId, input)
    if err != nil {
        logger.Error("Failed to create payment method",
            "orgId", authCtx.OrgId,
            "customerId", input.CustomerId,
            "error", err.Error())
        return mcp.NewToolResultError(fmt.Sprintf("Failed to create payment method: %s", err.Error())), nil
    }

    return mcp.NewToolResultText(fmt.Sprintf("Payment method created successfully. ID: %s, Type: %s", 
        paymentMethod.Id, paymentMethod.Type)), nil
}

// UpdatePaymentMethodHandler handles payment method updates
func UpdatePaymentMethodHandler(ctx context.Context, request mcp.CallToolRequest,
                               customerService interfaces.CustomerService,
                               authService *middleware.AuthService,
                               logger logger.Logger) (*mcp.CallToolResult, error) {

    // Extract authentication
    authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
    if err != nil {
        return mcp.NewToolResultError("Authentication required"), nil
    }

    // Parse input
    type UpdatePaymentMethodInput struct {
        CustomerID      string `json:"customer_id"`
        PaymentMethodID string `json:"payment_method_id"`
        dto.UpdatePaymentMethodInput
    }

    var input UpdatePaymentMethodInput
    if err := request.BindArguments(&input); err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
    }

    if input.CustomerID == "" || input.PaymentMethodID == "" {
        return mcp.NewToolResultError("customer_id and payment_method_id are required"), nil
    }

    // Log operation
    logger.Info("Updating payment method via MCP",
        "orgId", authCtx.OrgId,
        "userId", authCtx.User.Id,
        "customerId", input.CustomerID,
        "paymentMethodId", input.PaymentMethodID)

    // Call service
    paymentMethod, err := customerService.UpdatePaymentMethod(ctx, authCtx.OrgId, input.PaymentMethodID, input.UpdatePaymentMethodInput)
    if err != nil {
        logger.Error("Failed to update payment method",
            "orgId", authCtx.OrgId,
            "customerId", input.CustomerID,
            "paymentMethodId", input.PaymentMethodID,
            "error", err.Error())
        return mcp.NewToolResultError(fmt.Sprintf("Failed to update payment method: %s", err.Error())), nil
    }

    return mcp.NewToolResultText(fmt.Sprintf("Payment method updated successfully. ID: %s", 
        paymentMethod.Id)), nil
}

// GetPaymentMethodHandler handles payment method retrieval
func GetPaymentMethodHandler(ctx context.Context, request mcp.CallToolRequest,
                            customerService interfaces.CustomerService,
                            authService *middleware.AuthService,
                            logger logger.Logger) (*mcp.CallToolResult, error) {

    // Extract authentication
    authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
    if err != nil {
        return mcp.NewToolResultError("Authentication required"), nil
    }

    // Extract payment method ID using MCP helper
    paymentMethodId, err := request.RequireString("payment_method_id")
    if err != nil {
        return mcp.NewToolResultError("payment_method_id is required"), nil
    }

    // Log operation
    logger.Info("Retrieving payment method via MCP",
        "orgId", authCtx.OrgId,
        "userId", authCtx.User.Id,
        "paymentMethodId", paymentMethodId)

    // Call service
    paymentMethod, err := customerService.GetPaymentMethod(ctx, authCtx.OrgId, paymentMethodId)
    if err != nil {
        logger.Error("Failed to retrieve payment method",
            "orgId", authCtx.OrgId,
            "paymentMethodId", paymentMethodId,
            "error", err.Error())
        return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve payment method: %s", err.Error())), nil
    }

    return mcp.NewToolResultText(fmt.Sprintf("Payment method found - ID: %s, Type: %s, Customer: %s", 
        paymentMethod.Id, paymentMethod.Type, paymentMethod.CustomerId)), nil
}