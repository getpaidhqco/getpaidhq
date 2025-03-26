package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/mapper"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/interfaces"
	app_lib "payloop/internal/application/lib/authz"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/lib"
	"time"
)

// OrderController data type
type OrderController struct {
	service interfaces.OrderService
	logger  logger.Logger
	authz   app_lib.Authz
}

// NewOrderController creates new order controller
func NewOrderController(orderService interfaces.OrderService, logger logger.Logger, authz app_lib.Authz) OrderController {
	return OrderController{
		service: orderService,
		logger:  logger,
		authz:   authz,
	}
}

func (o OrderController) CreateOrder(c *gin.Context) {
	var input request.CreateOrderRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)

	allowed := o.authz.Enforce(authUser, app_lib.CreateOrder, "")
	if !allowed {
		apiErr := api.NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if input.SessionId == "" && len(input.Cart.Items) == 0 {
		apiErr := api.NewApiError(lib.ValidationError, "You must specify cart or session_id", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if len(input.Cart.Items) > 0 && input.Cart.Currency == "" {
		apiErr := api.NewApiError(lib.ValidationError, "Currency is required", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	rsp, err := o.service.CreateOrder(c.Request.Context(), orders.CreateOrderInput{
		OrgId:    authUser.OrgId,
		Currency: input.Cart.Currency,
		Customer: orders.CreateOrderCommandCustomer{
			Id:        input.Customer.ID,
			Email:     input.Customer.Email,
			FirstName: input.Customer.FirstName,
			LastName:  input.Customer.LastName,
			Phone:     input.Customer.Phone,
			Metadata:  nil,
		},
		SessionId:       input.SessionId,
		PaymentMethodId: input.PaymentMethodId,
		CartItems:       mapper.ToCartItems(input.Cart.Items),
		PspId:           common.Gateway(input.PspId),
		Metadata:        nil,
		Options:         input.Options,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, map[string]interface{}{
		"order": response.NewOrderFromEntity(rsp.Order),
		"psp":   rsp.Psp.PspResponse,
	})
}

func (o OrderController) CompleteOrder(c *gin.Context) {
	var input request.CompleteOrderRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	id := c.Param("id")

	allowed := o.authz.Enforce(authUser, app_lib.CreateOrder, "")
	if !allowed {
		apiErr := api.NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	var completedAt time.Time
	if input.Payment.CompletedAt != "" {
		parsed, err := time.Parse(time.RFC3339, input.Payment.CompletedAt)
		if err != nil {
			apiErr := api.NewApiError(lib.ValidationError, "Invalid completed_at format", nil)
			c.JSON(apiErr.GetHttpErrorCode(), apiErr)
			return
		}
		completedAt = parsed
	}

	rsp, err := o.service.CompleteOrder(c.Request.Context(), orders.CompleteOrderInput{
		OrgId:           authUser.OrgId,
		Id:              id,
		PaymentMethodId: input.PaymentMethodId,
		PaymentMethod: orders.CompleteOrderInputPaymentMethod{
			Psp:       input.PaymentMethod.Psp,
			Name:      input.PaymentMethod.Name,
			IsDefault: input.PaymentMethod.IsDefault,
			BillingAddress: orders.Address{
				FirstName:  input.PaymentMethod.BillingAddress.FirstName,
				LastName:   input.PaymentMethod.BillingAddress.LastName,
				Email:      input.PaymentMethod.BillingAddress.Email,
				Phone:      input.PaymentMethod.BillingAddress.Phone,
				Line1:      input.PaymentMethod.BillingAddress.Line1,
				Line2:      input.PaymentMethod.BillingAddress.Line2,
				City:       input.PaymentMethod.BillingAddress.City,
				State:      input.PaymentMethod.BillingAddress.State,
				PostalCode: input.PaymentMethod.BillingAddress.PostalCode,
				Country:    input.PaymentMethod.BillingAddress.Country,
			},
			Type:     input.PaymentMethod.Type,
			Token:    input.PaymentMethod.Token,
			Metadata: input.PaymentMethod.Metadata,
		},
		Payment: orders.CompleteOrderInputPayment{
			PspId:       input.Payment.PspId,
			CompletedAt: completedAt,
			Reference:   input.Payment.Reference,
			Amount:      input.Payment.Amount,
			Currency:    input.Payment.Currency,
			Metadata:    input.Payment.Metadata,
		},
		Metadata: input.Metadata,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewOrderFromEntity(rsp))
}

func (o OrderController) ListSubscriptions(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	id := c.Param("id")

	allowed := o.authz.Enforce(authUser, app_lib.ListOrderSubscriptions, "")
	if !allowed {
		apiErr := api.NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	rsp, err := o.service.ListOrderSubscriptions(c.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, rsp)
}

func (o OrderController) List(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	pagination := request.GetPagination(c)

	ords, total, err := o.service.List(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	var orderRsp []response.Order
	for _, order := range ords {
		orderRsp = append(orderRsp, response.NewOrderFromEntity(order))
	}

	c.JSON(200, response.ListResponse{
		Data: orderRsp,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

func (o OrderController) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	id := c.Param("id")

	order, err := o.service.FindById(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewOrderFromEntity(order))
}
