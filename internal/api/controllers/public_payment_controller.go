package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/api/mappers"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

type PublicPaymentController struct {
	paymentLinkService interfaces.PaymentLinkService
	orderService       interfaces.OrderService
	invoiceService     interfaces.InvoiceOrchestrationService
	customerService    interfaces.CustomerService
	orgService         services.OrgService
	logger             logger.Logger
}

func NewPublicPaymentController(
	paymentLinkService interfaces.PaymentLinkService,
	orderService interfaces.OrderService,
	invoiceService interfaces.InvoiceOrchestrationService,
	customerService interfaces.CustomerService,
	orgService services.OrgService,
	logger logger.Logger,
) PublicPaymentController {
	return PublicPaymentController{
		paymentLinkService: paymentLinkService,
		orderService:       orderService,
		invoiceService:     invoiceService,
		customerService:    customerService,
		orgService:         orgService,
		logger:             logger,
	}
}

// extractToken gets token from query parameter
func (c PublicPaymentController) extractToken(ctx *gin.Context) string {
	return ctx.Query("token")
}

// validatePaymentLinkAccess validates token access to a payment link
func (c PublicPaymentController) validatePaymentLinkAccess(ctx context.Context, slug, token string) (entities.PaymentLink, error) {
	// Use the service method for validation
	paymentLink, err := c.paymentLinkService.ValidatePaymentLinkAccess(ctx, slug, token)
	if err != nil {
		return entities.PaymentLink{}, err
	}

	return paymentLink, nil
}

// getOrgName fetches the organization name, falling back to orgId if fetch fails
func (c PublicPaymentController) getOrgName(ctx context.Context, orgId string) string {
	org, err := c.orgService.Get(ctx, orgId)
	if err != nil {
		c.logger.Warn("Failed to fetch organization details", "error", err, "orgId", orgId)
		// Fall back to orgId if we can't fetch the organization name
		return orgId
	}
	return org.Name
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

	linkType, ok := paymentLink.Data["type"].(string)
	if !ok {
		apiErr := api.NewApiError(lib.InternalError, "Payment link type not specified", nil)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Fetch organization name
	orgName := c.getOrgName(ctx.Request.Context(), paymentLink.OrgId)

	var responseData response.PublicPaymentDetailsResponse

	switch linkType {
	case "invoice":
		invoiceId, ok := paymentLink.Data["invoice_id"].(string)
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

		// Fetch customer data if customer ID is present in the invoice
		var customerData response.PublicCustomer
		if invoice.CustomerId != "" {
			customer, err := c.customerService.Get(ctx.Request.Context(), paymentLink.OrgId, invoice.CustomerId)
			if err != nil {
				c.logger.Warn("Failed to fetch customer details", "error", err, "customerId", invoice.CustomerId)
				// Continue without customer data rather than failing the whole request
			} else {
				customerResponse := response.NewPublicCustomerFromEntity(customer)
				customerData = customerResponse
			}
		}

		responseData = response.PublicPaymentDetailsResponse{
			Type:     "invoice",
			Invoice:  response.NewInvoiceFromEntity(invoice),
			Customer: customerData,
			Config:   paymentLink.Config,
			Org: response.PublicOrgResponse{
				Id:   paymentLink.OrgId,
				Name: orgName,
			},
		}

	case "checkout":
		responseData = response.PublicPaymentDetailsResponse{
			Type:   "checkout",
			Config: paymentLink.Config,
			Org: response.PublicOrgResponse{
				Id:   paymentLink.OrgId,
				Name: orgName,
			},
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

	linkType, ok := paymentLink.Data["type"].(string)
	if !ok {
		apiErr := api.NewApiError(lib.InternalError, "Payment link type not specified", nil)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Handle different payment link types
	switch linkType {
	case "invoice":
		invoiceId, ok := paymentLink.Data["invoice_id"].(string)
		if !ok {
			apiErr := api.NewApiError(lib.InternalError, "Invoice ID not found in payment link", nil)
			ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
			return
		}

		// Convert API request to application DTO
		appInput := mappers.ToCreateOrderFromInvoiceInput(input)

		// Call the new service method
		orderResponse, err := c.invoiceService.CreateOrderFromInvoice(
			ctx.Request.Context(),
			paymentLink.OrgId,
			invoiceId,
			appInput,
		)
		if err != nil {
			c.logger.Error("Failed to create order from invoice", "error", err, "invoiceId", invoiceId)

			apiErr := api.NewApiErrorFromError(err)
			ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
			return
		}

		publicOrderResponse := response.PublicOrderResponse{
			CreateOrderResponse: orderResponse,
		}
		// Convert to public response
		ctx.JSON(http.StatusOK, publicOrderResponse)

	default:
		apiErr := api.NewApiError(lib.BadRequestError, "Unsupported payment link type "+linkType, nil)
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
