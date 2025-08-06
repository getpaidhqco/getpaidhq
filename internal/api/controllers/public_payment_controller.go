package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/api/mappers"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

type PublicPaymentController struct {
	paymentLinkService interfaces.PaymentLinkService
	orderService       interfaces.OrderService
	invoiceService     interfaces.InvoiceService
	logger            logger.Logger
}

func NewPublicPaymentController(
	paymentLinkService interfaces.PaymentLinkService,
	orderService interfaces.OrderService,
	invoiceService interfaces.InvoiceService,
	logger logger.Logger,
) PublicPaymentController {
	return PublicPaymentController{
		paymentLinkService: paymentLinkService,
		orderService:      orderService,
		invoiceService:    invoiceService,
		logger:            logger,
	}
}

// extractToken gets token from query parameter
func (c PublicPaymentController) extractToken(ctx *gin.Context) string {
	return ctx.Query("token")
}

// validatePaymentLinkAccess validates token access to a payment link
func (c PublicPaymentController) validatePaymentLinkAccess(ctx context.Context, slug, token string) (*entities.PaymentLink, error) {
	// 1. Find payment link by slug
	paymentLink, err := c.paymentLinkService.GetPaymentLinkBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("payment link not found")
	}

	// TODO: Token validation will be implemented once PaymentLink entity is updated with TokenHash field
	// For now, we'll proceed without token validation as a placeholder

	// 2. Check payment link status (using string comparison for now)
	if paymentLink.Status != "active" {
		return nil, fmt.Errorf("payment link not active")
	}

	// 3. Check expiration (ExpiresAt is time.Time, check if zero value)
	if !paymentLink.ExpiresAt.IsZero() && time.Now().After(paymentLink.ExpiresAt) {
		return nil, fmt.Errorf("payment link expired")
	}

	// 4. Check single use (UsedAt is time.Time, check if zero value)
	if paymentLink.SingleUse && !paymentLink.UsedAt.IsZero() {
		return nil, fmt.Errorf("payment link already used")
	}

	return &paymentLink, nil
}

// GetPaymentDetails handles GET /api/pay/:slug
func (c PublicPaymentController) GetPaymentDetails(ctx *gin.Context) {
	slug := ctx.Param("slug")
	token := c.extractToken(ctx)

	// Validate access
	paymentLink, err := c.validatePaymentLinkAccess(ctx.Request.Context(), slug, token)
	if err != nil {
		apiErr := api.NewApiError(lib.AuthenticationError, err.Error(), nil)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Extract data based on payment link type - Data is stored as JSON bytes
	var data map[string]interface{}
	if err := json.Unmarshal(paymentLink.Data, &data); err != nil {
		apiErr := api.NewApiError(lib.InternalError, "Invalid payment link data", err.Error())
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	linkType, ok := data["type"].(string)
	if !ok {
		apiErr := api.NewApiError(lib.InternalError, "Payment link type not specified", nil)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	var responseData response.PublicPaymentDetailsResponse

	switch linkType {
	case "invoice":
		invoiceId, ok := data["invoiceId"].(string)
		if !ok {
			apiErr := api.NewApiError(lib.InternalError, "Invoice ID not found in payment link", nil)
			ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
			return
		}

		invoice, err := c.invoiceService.Get(ctx.Request.Context(), paymentLink.OrgId, invoiceId)
		if err != nil {
			c.logger.Error("Failed to fetch invoice", "error", err, "invoiceId", invoiceId)
			apiErr := api.NewApiError(lib.InternalError, "Failed to fetch invoice details", err.Error())
			ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
			return
		}

		responseData = response.PublicPaymentDetailsResponse{
			Type:          "invoice",
			Invoice:       mappers.ToPublicInvoiceResponse(invoice),
			PaymentConfig: paymentLink.Config,
			OrgId:         paymentLink.OrgId,
		}

	case "checkout":
		responseData = response.PublicPaymentDetailsResponse{
			Type:          "checkout",
			CheckoutItems: data["items"],
			PaymentConfig: paymentLink.Config,
			OrgId:         paymentLink.OrgId,
		}

	default:
		apiErr := api.NewApiError(lib.BadRequestError, "Unsupported payment link type", nil)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.JSON(http.StatusOK, responseData)
}

// CreateOrder handles POST /api/pay/:slug/create-order
func (c PublicPaymentController) CreateOrder(ctx *gin.Context) {
	slug := ctx.Param("slug")
	token := c.extractToken(ctx)

	var input request.PublicCreateOrderRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiError(lib.BadRequestError, "Invalid request format", err.Error())
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Validate access
	paymentLink, err := c.validatePaymentLinkAccess(ctx.Request.Context(), slug, token)
	if err != nil {
		apiErr := api.NewApiError(lib.AuthenticationError, err.Error(), nil)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Extract payment link data - Data is stored as JSON bytes
	var data map[string]interface{}
	if err := json.Unmarshal(paymentLink.Data, &data); err != nil {
		apiErr := api.NewApiError(lib.InternalError, "Invalid payment link data", err.Error())
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	linkType, ok := data["type"].(string)
	if !ok {
		apiErr := api.NewApiError(lib.InternalError, "Payment link type not specified", nil)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Handle different payment link types
	switch linkType {
	case "invoice":
		invoiceId, ok := data["invoiceId"].(string)
		if !ok {
			apiErr := api.NewApiError(lib.InternalError, "Invoice ID not found in payment link", nil)
			ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
			return
		}

		// Use existing InvoiceService.InitiatePayment method
		var billingAddress entities.Address
		if input.BillingAddress != nil {
			billingAddress = *input.BillingAddress
		}

		appInput := dto.InitiatePaymentInput{
			PaymentProcessor: input.PaymentProcessor,
			BillingAddress:   billingAddress,
			SuccessUrl:      input.SuccessUrl,
			CancelUrl:       input.CancelUrl,
			Metadata:        input.Metadata,
		}

		order, orderResponse, err := c.invoiceService.InitiatePayment(
			ctx.Request.Context(),
			paymentLink.OrgId,
			invoiceId,
			appInput,
		)
		if err != nil {
			c.logger.Error("Failed to create order", "error", err, "invoiceId", invoiceId)
			apiErr := api.NewApiError(lib.InternalError, "Failed to create order", err.Error())
			ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
			return
		}

		// Mark payment link as used if single-use
		if paymentLink.SingleUse {
			// Note: PaymentLinkService would need a MarkAsUsed method
			c.logger.Info("Single-use payment link used", "paymentLinkId", paymentLink.Id)
		}

		// Convert to public response
		publicResponse := mappers.ToPublicOrderResponse(order, orderResponse, input.PaymentProcessor)
		ctx.JSON(http.StatusOK, publicResponse)

	default:
		apiErr := api.NewApiError(lib.BadRequestError, "Unsupported payment link type", nil)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
}

// GetOrderStatus handles GET /api/pay/:slug/order/:orderId/status
func (c PublicPaymentController) GetOrderStatus(ctx *gin.Context) {
	slug := ctx.Param("slug")
	orderId := ctx.Param("orderId")
	token := c.extractToken(ctx)

	// Validate access
	paymentLink, err := c.validatePaymentLinkAccess(ctx.Request.Context(), slug, token)
	if err != nil {
		apiErr := api.NewApiError(lib.AuthenticationError, err.Error(), nil)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Get order and verify it belongs to this organization
	order, err := c.orderService.FindById(ctx.Request.Context(), paymentLink.OrgId, orderId)
	if err != nil {
		apiErr := api.NewApiError(lib.NotFoundError, "Order not found", err.Error())
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert to public response
	statusResponse := response.PublicOrderStatusResponse{
		OrderId:  order.Id,
		Status:   string(order.Status),
		Amount:   int(order.Total),
		Currency: order.Currency,
	}

	ctx.JSON(http.StatusOK, statusResponse)
}
