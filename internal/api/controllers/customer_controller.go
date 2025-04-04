package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
)

type CustomerController struct {
	customerService interfaces.CustomerService
	logger          logger.Logger
}

// NewUserController creates new user controller
func NewCustomerController(customerService interfaces.CustomerService, logger logger.Logger) CustomerController {
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

// CreateCustomerPaymentMethod handles the creation of a new payment method for a customer
func (cc CustomerController) UpdateCustomerPaymentMethod(c *gin.Context) {
	var input request.UpdatePaymentMethodRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	customerId := c.Param("id")
	pmId := c.Param("pmid")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	paymentMethod, err := cc.customerService.UpdatePaymentMethod(
		c.Request.Context(), authUser.OrgId, interfaces.UpdatePaymentMethodInput{
			UpdatePaymentMethodRequest: input,
			PaymentMethodId:            pmId,
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

func (cc CustomerController) GetCustomerPaymentMethod(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	paymentMethodId := c.Param("id")

	paymentMethod, err := cc.customerService.GetPaymentMethod(c.Request.Context(), authUser.OrgId, paymentMethodId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}
