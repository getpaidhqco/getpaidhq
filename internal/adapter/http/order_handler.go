package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/core/service"
	"payloop/internal/lib"
)

// OrderHandler handles HTTP requests for orders.
type OrderHandler struct {
	service *service.OrderService
	logger  port.Logger
	authz   port.Authz
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(orderService *service.OrderService, logger port.Logger, authz port.Authz) *OrderHandler {
	return &OrderHandler{
		service: orderService,
		logger:  logger,
		authz:   authz,
	}
}

// RegisterRoutes registers order routes on the given router group.
func (o *OrderHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/orders", o.CreateOrder)
	rg.POST("/orders/:id/complete", o.CompleteOrder)
	rg.GET("/orders/:id", o.Get)
	rg.GET("/orders", o.List)
	rg.GET("/orders/:id/subscriptions", o.ListSubscriptions)
}

func (o *OrderHandler) CreateOrder(c *gin.Context) {
	var input CreateOrderRequest
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}

	allowed := o.authz.Enforce(authUser, port.ActionCreateOrder, "")
	if !allowed {
		apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if input.SessionId == "" && len(input.Cart.Items) == 0 {
		apiErr := NewApiError(lib.ValidationError, "You must specify cart or session_id", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if len(input.Cart.Items) > 0 && input.Cart.Currency == "" {
		apiErr := NewApiError(lib.ValidationError, "Currency is required", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	rsp, err := o.service.CreateOrder(c.Request.Context(), domain.CreateOrderInput{
		OrgId:    authUser.OrgId,
		Currency: domain.Currency(input.Cart.Currency),
		Customer: domain.CreateOrderCommandCustomer{
			Id:        input.Customer.ID,
			Email:     input.Customer.Email,
			FirstName: input.Customer.FirstName,
			LastName:  input.Customer.LastName,
			Phone:     input.Customer.Phone,
			Metadata:  nil,
		},
		SessionId:       input.SessionId,
		PaymentMethodId: input.PaymentMethodId,
		CartItems:       ToCartItems(input.Cart.Items),
		PspId:           domain.Gateway(input.PspId),
		Metadata:        nil,
		Options:         input.Options,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, map[string]interface{}{
		"order": NewOrderFromEntity(rsp.Order),
		"psp":   rsp.Psp.PspResponse,
	})
}

func (o *OrderHandler) CompleteOrder(c *gin.Context) {
	var input CompleteOrderRequest
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
	id := c.Param("id")

	allowed := o.authz.Enforce(authUser, port.ActionCreateOrder, "")
	if !allowed {
		apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	var completedAt time.Time
	if input.Payment.CompletedAt != "" {
		parsed, err := time.Parse(time.RFC3339, input.Payment.CompletedAt)
		if err != nil {
			apiErr := NewApiError(lib.ValidationError, "Invalid completed_at format", nil)
			c.JSON(apiErr.GetHttpErrorCode(), apiErr)
			return
		}
		completedAt = parsed
	}

	rsp, err := o.service.CompleteOrder(c.Request.Context(), domain.CompleteOrderInput{
		OrgId:           authUser.OrgId,
		Id:              id,
		PaymentMethodId: input.PaymentMethodId,
		PaymentMethod: domain.CompleteOrderInputPaymentMethod{
			Psp:       input.PaymentMethod.Psp,
			Name:      input.PaymentMethod.Name,
			IsDefault: input.PaymentMethod.IsDefault,
			Details:   input.PaymentMethod.Details,
			BillingAddress: domain.Address{
				FirstName:  input.PaymentMethod.BillingAddress.FirstName,
				LastName:   input.PaymentMethod.BillingAddress.LastName,
				Email:      input.PaymentMethod.BillingAddress.Email,
				Phone:      input.PaymentMethod.BillingAddress.Phone,
				Line1:      input.PaymentMethod.BillingAddress.Line1,
				Line2:      input.PaymentMethod.BillingAddress.Line2,
				City:       input.PaymentMethod.BillingAddress.City,
				State:      input.PaymentMethod.BillingAddress.State,
				PostalCode: input.PaymentMethod.BillingAddress.PostalCode,
				Country:    domain.Country(input.PaymentMethod.BillingAddress.Country),
			},
			Type:     domain.PaymentMethodType(input.PaymentMethod.Type),
			Token:    input.PaymentMethod.Token,
			Metadata: input.PaymentMethod.Metadata,
		},
		Payment: domain.CompleteOrderInputPayment{
			PspId:       input.Payment.PspId,
			CompletedAt: completedAt,
			Reference:   input.Payment.Reference,
			Amount:      input.Payment.Amount,
			Currency:    domain.Currency(input.Payment.Currency),
			Metadata:    input.Payment.Metadata,
		},
		Metadata: input.Metadata,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, NewOrderFromEntity(rsp))
}

func (o *OrderHandler) ListSubscriptions(c *gin.Context) {
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
	id := c.Param("id")

	allowed := o.authz.Enforce(authUser, port.ActionListOrderSubscriptions, "")
	if !allowed {
		apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	rsp, err := o.service.ListOrderSubscriptions(c.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, rsp)
}

func (o *OrderHandler) List(c *gin.Context) {
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
	orgId := authUser.OrgId
	pagination := GetPagination(c)

	ords, total, err := o.service.List(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	var orderRsp = make([]OrderResponse, 0, len(ords))
	for _, order := range ords {
		orderRsp = append(orderRsp, NewOrderFromEntity(order))
	}

	c.JSON(200, ListResponse{
		Data: orderRsp,
		Meta: Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

func (o *OrderHandler) Get(c *gin.Context) {
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
	orgId := authUser.OrgId
	id := c.Param("id")

	order, err := o.service.FindById(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, NewOrderFromEntity(order))
}
