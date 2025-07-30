package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/mappers"
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

	// Convert API DTO to application DTO
	appInput := mappers.ToCreateCustomerInput(input)

	customer, err := cc.customerService.Create(c.Request.Context(), authUser.OrgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entity to API response
	response := mappers.ToCustomerResponse(customer)
	c.JSON(http.StatusCreated, response)
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

	// Convert API DTO to application DTO
	appInput := mappers.ToCreatePaymentMethodInput(input, customerId)

	paymentMethod, err := cc.customerService.CreatePaymentMethod(c.Request.Context(), authUser.OrgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}

// UpdateCustomerPaymentMethod handles the update of a payment method for a customer
func (cc CustomerController) UpdateCustomerPaymentMethod(c *gin.Context) {
	var input request.UpdatePaymentMethodRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	pmId := c.Param("pmid")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert API DTO to application DTO
	appInput := mappers.ToUpdatePaymentMethodInput(input)

	paymentMethod, err := cc.customerService.UpdatePaymentMethod(c.Request.Context(), authUser.OrgId, pmId, appInput)
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

// Get handles retrieving a customer by ID
func (cc CustomerController) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	customerId := c.Param("id")

	customer, err := cc.customerService.Get(c.Request.Context(), authUser.OrgId, customerId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entity to API response
	response := mappers.ToCustomerResponse(customer)
	c.JSON(http.StatusOK, response)
}

// List handles retrieving a list of customers with pagination and search
func (cc CustomerController) List(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	pagination := request.GetPagination(c)

	// Convert API DTO to application DTO
	appPagination := mappers.ToPagination(pagination)

	result, err := cc.customerService.List(c.Request.Context(), authUser.OrgId, appPagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert paginated result to API response
	response := mappers.ToCustomerListResponse(result)
	c.JSON(http.StatusOK, response)
}
