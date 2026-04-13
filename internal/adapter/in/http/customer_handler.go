package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/port"
	"payloop/internal/application/interfaces"
)

// CustomerHandler handles HTTP requests for customers.
type CustomerHandler struct {
	customerService interfaces.CustomerService
	logger          port.Logger
	authz           port.Authz
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(customerService interfaces.CustomerService, logger port.Logger, authz port.Authz) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
		logger:          logger,
		authz:           authz,
	}
}

// RegisterRoutes registers customer routes on the given router group.
func (cc *CustomerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/customers", cc.List)
	rg.GET("/customers/:id", cc.Get)
	rg.POST("/customers", cc.Create)
	rg.POST("/customers/:id/payment-methods", cc.checkAuthz(port.ActionCreatePaymentMethod), cc.CreateCustomerPaymentMethod)
	rg.PUT("/customers/:id/payment-methods/:pmid", cc.checkAuthz(port.ActionCreatePaymentMethod), cc.UpdateCustomerPaymentMethod)
}

func (cc *CustomerHandler) checkAuthz(action port.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		authUser := user.(port.AuthUser)
		allowed := cc.authz.Enforce(authUser, action, "")
		if !allowed {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Create handles the creation of a new customer.
func (cc *CustomerHandler) Create(c *gin.Context) {
	var input CreateCustomerRequest
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)

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

// CreateCustomerPaymentMethod handles the creation of a new payment method for a customer.
func (cc *CustomerHandler) CreateCustomerPaymentMethod(c *gin.Context) {
	var input CreatePaymentMethodRequest
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	customerId := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	paymentMethod, err := cc.customerService.CreatePaymentMethod(
		c.Request.Context(), authUser.OrgId, CreatePaymentMethodInput{
			CreatePaymentMethodRequest: input,
			OrgId:                      authUser.OrgId,
			CustomerId:                 customerId,
		})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}

// UpdateCustomerPaymentMethod handles updating a payment method for a customer.
func (cc *CustomerHandler) UpdateCustomerPaymentMethod(c *gin.Context) {
	var input UpdatePaymentMethodRequest
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	customerId := c.Param("id")
	pmId := c.Param("pmid")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	paymentMethod, err := cc.customerService.UpdatePaymentMethod(
		c.Request.Context(), authUser.OrgId, UpdatePaymentMethodInput{
			UpdatePaymentMethodRequest: input,
			PaymentMethodId:            pmId,
			OrgId:                      authUser.OrgId,
			CustomerId:                 customerId,
		})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}

// GetCustomerPaymentMethod handles retrieving a payment method by ID.
func (cc *CustomerHandler) GetCustomerPaymentMethod(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	paymentMethodId := c.Param("id")

	paymentMethod, err := cc.customerService.GetPaymentMethod(c.Request.Context(), authUser.OrgId, paymentMethodId)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, paymentMethod)
}

// Get handles retrieving a customer by ID.
func (cc *CustomerHandler) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	customerId := c.Param("id")

	customer, err := cc.customerService.Get(c.Request.Context(), authUser.OrgId, customerId)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, NewCustomerFromEntity(customer))
}

// List handles retrieving a list of customers with pagination and search.
func (cc *CustomerHandler) List(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
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
