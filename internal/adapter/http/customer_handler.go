package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/core/service"
)

type CustomerHandler struct {
	customerService *service.CustomerService
	logger          port.Logger
	authz           port.Authz
}

func NewCustomerHandler(customerService *service.CustomerService, logger port.Logger, authz port.Authz) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
		logger:          logger,
		authz:           authz,
	}
}

func (cc *CustomerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/customers", cc.List)
	rg.GET("/customers/:id", cc.Get)
	rg.POST("/customers", cc.Create)
	rg.POST("/customers/:id/payment-methods", cc.CreateCustomerPaymentMethod)
	rg.PUT("/customers/:id/payment-methods/:pmid", cc.UpdateCustomerPaymentMethod)
}

func (cc *CustomerHandler) Create(c *gin.Context) {
	var input domain.CreateCustomerInput
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	customer, err := cc.customerService.Create(c.Request.Context(), authUser.OrgId, input)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, customer)
}

func (cc *CustomerHandler) CreateCustomerPaymentMethod(c *gin.Context) {
	var input domain.CreatePaymentMethodInput
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
	customerId := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	input.OrgId = authUser.OrgId
	input.CustomerId = customerId

	paymentMethod, err := cc.customerService.CreatePaymentMethod(c.Request.Context(), authUser.OrgId, input)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}

func (cc *CustomerHandler) UpdateCustomerPaymentMethod(c *gin.Context) {
	var input domain.UpdatePaymentMethodInput
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
	customerId := c.Param("id")
	pmId := c.Param("pmid")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	input.OrgId = authUser.OrgId
	input.CustomerId = customerId
	input.PaymentMethodId = pmId

	paymentMethod, err := cc.customerService.UpdatePaymentMethod(c.Request.Context(), authUser.OrgId, input)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}

func (cc *CustomerHandler) GetCustomerPaymentMethod(c *gin.Context) {
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
	paymentMethodId := c.Param("id")

	paymentMethod, err := cc.customerService.GetPaymentMethod(c.Request.Context(), authUser.OrgId, paymentMethodId)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}

func (cc *CustomerHandler) Get(c *gin.Context) {
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
	customerId := c.Param("id")

	customer, err := cc.customerService.Get(c.Request.Context(), authUser.OrgId, customerId)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, NewCustomerFromEntity(customer))
}

func (cc *CustomerHandler) List(c *gin.Context) {
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
	pagination := GetPagination(c)

	customers, total, err := cc.customerService.List(c.Request.Context(), authUser.OrgId, pagination)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	customerResponses := make([]CustomerResponse, len(customers))
	for i, customer := range customers {
		customerResponses[i] = NewCustomerFromEntity(customer)
	}

	c.JSON(http.StatusOK, ListResponse{
		Data: customerResponses,
		Meta: Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}
