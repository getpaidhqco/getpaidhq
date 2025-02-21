package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/interfaces"
	app_lib "payloop/internal/application/lib/authz"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/lib"
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

	order, psp, err := o.service.CreateOrderFromCart(c.Request.Context(), orders.CreateOrderInput{
		OrgId: authUser.OrgId,
		Customer: orders.CreateOrderCommandCustomer{
			ID:        input.Customer.ID,
			Email:     input.Customer.Email,
			FirstName: input.Customer.FirstName,
			LastName:  input.Customer.LastName,
			Phone:     input.Customer.Phone,
			Metadata:  nil,
		},
		CartId:    input.CartId,
		CartItems: nil,
		PspId:     common.Gateway(input.PspId),
		Metadata:  nil,
		Options:   input.Options,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, map[string]interface{}{
		"order": response.NewOrderFromEntity(order),
		"psp":   psp.PspResponse,
	})
}
