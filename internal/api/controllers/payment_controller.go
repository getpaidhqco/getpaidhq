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
	"payloop/internal/lib"
)

// PaymentController data type
type PaymentController struct {
	paymentService interfaces.PaymentService
	logger         logger.Logger
	authz          app_lib.Authz
}

// NewPaymentController creates new payment controller
func NewPaymentController(paymentService interfaces.PaymentService, logger logger.Logger, authz app_lib.Authz) PaymentController {
	return PaymentController{
		paymentService: paymentService,
		logger:         logger,
		authz:          authz,
	}
}

// Get retrieves a payment by ID
func (s PaymentController) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	id := c.Param("id")

	payment, err := s.paymentService.FindById(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewPaymentFromEntity(payment))
}

// List retrieves a list of payments
func (s PaymentController) List(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	pagination := request.GetPagination(c)

	payments, total, err := s.paymentService.List(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	paymentResponses := make([]response.Payment, len(payments))
	for i, payment := range payments {
		paymentResponses[i] = response.NewPaymentFromEntity(payment)
	}

	c.JSON(200, response.ListResponse{
		Data: paymentResponses,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// Refund creates a refund for a payment
func (s PaymentController) Refund(c *gin.Context) {
	var input request.RefundPaymentRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	id := c.Param("id")

	allowed := s.authz.Enforce(authUser, app_lib.RefundPayment, "")
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

	refund, err := s.paymentService.Refund(c.Request.Context(), orgId, id, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewRefundFromEntity(refund))
}
