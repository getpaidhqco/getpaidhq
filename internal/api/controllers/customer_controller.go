package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
)

type CustomerController struct {
	customerService services.CustomerService
	logger          logger.Logger
}

// NewUserController creates new user controller
func NewCustomerController(customerService services.CustomerService, logger logger.Logger) CustomerController {
	return CustomerController{
		customerService: customerService,
		logger:          logger,
	}
}

// Create handles the creation of a new customer
func (cc CustomerController) Create(c *gin.Context) {
	var input request.CreateCustomerRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	customer, err := cc.customerService.Create(c.Request.Context(), authUser.OrgId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, customer)
}

// CreateCustomerPaymentMethod handles the creation of a new payment method for a customer
func (cc CustomerController) CreateCustomerPaymentMethod(c *gin.Context) {
	var input request.CreatePaymentMethodRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	customerId := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	paymentMethod, err := cc.customerService.CreatePaymentMethod(
		c.Request.Context(), authUser.OrgId, interfaces.CreatePaymentMethodInput{
			CreatePaymentMethodRequest: input,
			OrgId:                      authUser.OrgId,
			CustomerId:                 customerId,
		})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}
