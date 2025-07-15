package handlers

import (
	"context"
	"fmt"
	"strconv"
	
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/mcp/middleware"
	"payloop/internal/mcp/schema"

	"github.com/mark3labs/mcp-go/mcp"
)

var productSchemaGenerator = schema.NewGenerator()

// CreateProductHandler handles product creation requests
func CreateProductHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Parse input
	var input dto.CreateProductInput
	if err := request.BindArguments(&input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
	}

	// Log operation
	logger.Info("Creating product via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"productName", input.Name)

	// Call service
	product, err := productService.CreateProduct(ctx, authCtx.OrgId, input)
	if err != nil {
		logger.Error("Failed to create product",
			"orgId", authCtx.OrgId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create product: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Product created successfully. ID: %s, Name: %s", 
		product.Id, product.Name)), nil
}

// GetProductHandler handles product retrieval requests
func GetProductHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Extract product ID
	productId, err := request.RequireString("product_id")
	if err != nil {
		return mcp.NewToolResultError("product_id is required"), nil
	}

	// Log operation
	logger.Info("Retrieving product via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"productId", productId)

	// Call service
	product, err := productService.FindById(ctx, authCtx.OrgId, productId)
	if err != nil {
		logger.Error("Failed to retrieve product",
			"orgId", authCtx.OrgId,
			"productId", productId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve product: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Product found - ID: %s, Name: %s, Description: %s",
		product.Id, product.Name, product.Description)), nil
}

// ListProductsHandler handles product listing requests
func ListProductsHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Parse pagination
	page := 1
	limit := 20
	
	if pageVal, err := request.RequireString("page"); err == nil {
		if p, err := strconv.Atoi(pageVal); err == nil && p > 0 {
			page = p
		}
	}
	
	if limitVal, err := request.RequireString("limit"); err == nil {
		if l, err := strconv.Atoi(limitVal); err == nil && l > 0 {
			limit = l
		}
	}

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Log operation
	logger.Info("Listing products via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"page", page,
		"limit", limit)

	// Call service with dto.Pagination which the interface expects
	pagination := dto.Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
	products, total, err := productService.List(ctx, authCtx.OrgId, pagination)
	if err != nil {
		logger.Error("Failed to list products",
			"orgId", authCtx.OrgId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list products: %s", err.Error())), nil
	}

	// Format response
	if len(products) == 0 {
		return mcp.NewToolResultText("No products found"), nil
	}

	responseText := fmt.Sprintf("Found %d products (total: %d, page: %d, limit: %d):\n",
		len(products), total, page, limit)

	for i, product := range products {
		responseText += fmt.Sprintf("%d. ID: %s, Name: %s\n",
			i+1, product.Id, product.Name)
	}

	return mcp.NewToolResultText(responseText), nil
}

// UpdateProductHandler handles product update requests
func UpdateProductHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Parse input
	type UpdateProductInput struct {
		ProductID string `json:"product_id"`
		dto.UpdateProductInput
	}

	var input UpdateProductInput
	if err := request.BindArguments(&input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
	}

	if input.ProductID == "" {
		return mcp.NewToolResultError("product_id is required"), nil
	}

	// Log operation
	logger.Info("Updating product via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"productId", input.ProductID)

	// Call service
	product, err := productService.UpdateProduct(ctx, authCtx.OrgId, input.ProductID, input.UpdateProductInput)
	if err != nil {
		logger.Error("Failed to update product",
			"orgId", authCtx.OrgId,
			"productId", input.ProductID,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update product: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Product updated successfully. ID: %s, Name: %s",
		product.Id, product.Name)), nil
}

// DeleteProductHandler handles product deletion requests
func DeleteProductHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Extract product ID
	productId, err := request.RequireString("product_id")
	if err != nil {
		return mcp.NewToolResultError("product_id is required"), nil
	}

	// Log operation
	logger.Info("Deleting product via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"productId", productId)

	// Call service
	err = productService.DeleteProduct(ctx, authCtx.OrgId, productId)
	if err != nil {
		logger.Error("Failed to delete product",
			"orgId", authCtx.OrgId,
			"productId", productId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete product: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Product deleted successfully. ID: %s", productId)), nil
}

// CreateVariantHandler handles variant creation requests
func CreateVariantHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Parse input
	var input dto.CreateVariantInput
	if err := request.BindArguments(&input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
	}

	// Log operation
	logger.Info("Creating variant via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"productId", input.ProductID,
		"variantName", input.Name)

	// Call service with dto which the interface expects 
	variant, err := productService.CreateVariant(ctx, authCtx.OrgId, input.ProductID, input)
	if err != nil {
		logger.Error("Failed to create variant",
			"orgId", authCtx.OrgId,
			"productId", input.ProductID,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create variant: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Variant created successfully. ID: %s, Name: %s",
		variant.Id, variant.Name)), nil
}

// GetVariantHandler handles variant retrieval requests
func GetVariantHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Extract variant ID
	variantId, err := request.RequireString("variant_id")
	if err != nil {
		return mcp.NewToolResultError("variant_id is required"), nil
	}

	// Log operation
	logger.Info("Retrieving variant via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"variantId", variantId)

	// Call service
	variant, err := productService.GetVariant(ctx, authCtx.OrgId, variantId)
	if err != nil {
		logger.Error("Failed to retrieve variant",
			"orgId", authCtx.OrgId,
			"variantId", variantId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve variant: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Variant found - ID: %s, Name: %s, Product: %s",
		variant.Id, variant.Name, variant.ProductId)), nil
}

// ListVariantsHandler handles variant listing requests
func ListVariantsHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Parse input
	var input dto.ListVariantsInput
	if err := request.BindArguments(&input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
	}

	// Set defaults
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}

	// Log operation
	logger.Info("Listing variants via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"productId", input.ProductID,
		"page", input.Page,
		"limit", input.Limit)

	// Call service
	variants, total, err := productService.ListVariants(ctx, authCtx.OrgId, input.ProductID, dto.Pagination{
		Page:   input.Page,
		Limit:  input.Limit,
		Offset: (input.Page - 1) * input.Limit,
	})
	if err != nil {
		logger.Error("Failed to list variants",
			"orgId", authCtx.OrgId,
			"productId", input.ProductID,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list variants: %s", err.Error())), nil
	}

	// Format response
	if len(variants) == 0 {
		return mcp.NewToolResultText("No variants found"), nil
	}

	responseText := fmt.Sprintf("Found %d variants (total: %d, page: %d, limit: %d):\n",
		len(variants), total, input.Page, input.Limit)

	for i, variant := range variants {
		responseText += fmt.Sprintf("%d. ID: %s, Name: %s\n",
			i+1, variant.Id, variant.Name)
	}

	return mcp.NewToolResultText(responseText), nil
}

// UpdateVariantHandler handles variant update requests
func UpdateVariantHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Parse input
	var input dto.UpdateVariantInput
	if err := request.BindArguments(&input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
	}

	if input.VariantID == "" {
		return mcp.NewToolResultError("variant_id is required"), nil
	}

	// Log operation
	logger.Info("Updating variant via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"variantId", input.VariantID)

	// Call service
	variant, err := productService.UpdateVariant(ctx, authCtx.OrgId, input.VariantID, input)
	if err != nil {
		logger.Error("Failed to update variant",
			"orgId", authCtx.OrgId,
			"variantId", input.VariantID,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update variant: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Variant updated successfully. ID: %s, Name: %s",
		variant.Id, variant.Name)), nil
}

// DeleteVariantHandler handles variant deletion requests
func DeleteVariantHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Extract variant ID
	variantId, err := request.RequireString("variant_id")
	if err != nil {
		return mcp.NewToolResultError("variant_id is required"), nil
	}

	// Log operation
	logger.Info("Deleting variant via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"variantId", variantId)

	// Call service
	err = productService.DeleteVariant(ctx, authCtx.OrgId, variantId)
	if err != nil {
		logger.Error("Failed to delete variant",
			"orgId", authCtx.OrgId,
			"variantId", variantId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete variant: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Variant deleted successfully. ID: %s", variantId)), nil
}

// CreatePriceHandler handles price creation requests
func CreatePriceHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Parse input
	var input dto.CreatePriceInput
	if err := request.BindArguments(&input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
	}

	// Log operation
	logger.Info("Creating price via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"variantId", input.VariantId,
		"priceLabel", input.Label)

	// Call service
	price, err := productService.CreateProductPrice(ctx, input)
	if err != nil {
		logger.Error("Failed to create price",
			"orgId", authCtx.OrgId,
			"variantId", input.VariantId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create price: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Price created successfully. ID: %s, Label: %s",
		price.Id, price.Label)), nil
}

// GetPriceHandler handles price retrieval requests
func GetPriceHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Extract price ID
	priceId, err := request.RequireString("price_id")
	if err != nil {
		return mcp.NewToolResultError("price_id is required"), nil
	}

	// Log operation
	logger.Info("Retrieving price via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"priceId", priceId)

	// Call service
	price, err := productService.GetPrice(ctx, authCtx.OrgId, priceId)
	if err != nil {
		logger.Error("Failed to retrieve price",
			"orgId", authCtx.OrgId,
			"priceId", priceId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve price: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Price found - ID: %s, Label: %s, Unit Price: %d %s",
		price.Id, price.Label, price.UnitPrice, price.Currency)), nil
}

// ListPricesHandler handles price listing requests
func ListPricesHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Parse input
	var input dto.ListPricesInput
	if err := request.BindArguments(&input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
	}

	// Set defaults
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}

	// Log operation
	logger.Info("Listing prices via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"variantId", input.VariantId,
		"page", input.Page,
		"limit", input.Limit)

	// Call service
	prices, total, err := productService.ListPrices(ctx, authCtx.OrgId, input.VariantId, dto.Pagination{
		Page:   input.Page,
		Limit:  input.Limit,
		Offset: (input.Page - 1) * input.Limit,
	})
	if err != nil {
		logger.Error("Failed to list prices",
			"orgId", authCtx.OrgId,
			"variantId", input.VariantId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list prices: %s", err.Error())), nil
	}

	// Format response
	if len(prices) == 0 {
		return mcp.NewToolResultText("No prices found"), nil
	}

	responseText := fmt.Sprintf("Found %d prices (total: %d, page: %d, limit: %d):\n",
		len(prices), total, input.Page, input.Limit)

	for i, price := range prices {
		responseText += fmt.Sprintf("%d. ID: %s, Label: %s, Unit Price: %d %s\n",
			i+1, price.Id, price.Label, price.UnitPrice, price.Currency)
	}

	return mcp.NewToolResultText(responseText), nil
}

// UpdatePriceHandler handles price update requests
func UpdatePriceHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Parse input
	var input dto.UpdatePriceInput
	if err := request.BindArguments(&input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %s", err.Error())), nil
	}

	if input.PriceID == "" {
		return mcp.NewToolResultError("price_id is required"), nil
	}

	// Log operation
	logger.Info("Updating price via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"priceId", input.PriceID)

	// Call service
	price, err := productService.UpdatePrice(ctx, authCtx.OrgId, input.PriceID, input)
	if err != nil {
		logger.Error("Failed to update price",
			"orgId", authCtx.OrgId,
			"priceId", input.PriceID,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update price: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Price updated successfully. ID: %s, Label: %s",
		price.Id, price.Label)), nil
}

// DeletePriceHandler handles price deletion requests
func DeletePriceHandler(ctx context.Context, request mcp.CallToolRequest,
	productService interfaces.ProductService,
	authService *middleware.AuthService,
	logger logger.Logger) (*mcp.CallToolResult, error) {

	// Extract authentication
	authCtx, err := authService.ExtractAuthFromMCPRequest(ctx, request.GetArguments())
	if err != nil {
		return mcp.NewToolResultError("Authentication required"), nil
	}

	// Extract price ID
	priceId, err := request.RequireString("price_id")
	if err != nil {
		return mcp.NewToolResultError("price_id is required"), nil
	}

	// Log operation
	logger.Info("Deleting price via MCP",
		"orgId", authCtx.OrgId,
		"userId", authCtx.User.Id,
		"priceId", priceId)

	// Call service
	err = productService.DeletePrice(ctx, authCtx.OrgId, priceId)
	if err != nil {
		logger.Error("Failed to delete price",
			"orgId", authCtx.OrgId,
			"priceId", priceId,
			"error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete price: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Price deleted successfully. ID: %s", priceId)), nil
}
