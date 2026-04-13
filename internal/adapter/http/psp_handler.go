package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/core/service"
	"payloop/internal/lib"
)

// PspHandler handles HTTP requests for payment service providers.
type PspHandler struct {
	gatewayService *service.PspService
	logger         port.Logger
	authz          port.Authz
}

// NewPspHandler creates a new PspHandler.
func NewPspHandler(gatewayService *service.PspService, logger port.Logger, authz port.Authz) *PspHandler {
	return &PspHandler{
		gatewayService: gatewayService,
		logger:         logger,
		authz:          authz,
	}
}

// RegisterRoutes registers PSP routes on the given router group.
func (s *PspHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/gateways", s.checkAuthz(port.ActionCreatePaymentServiceProvider), s.Create)
}

func (s *PspHandler) checkAuthz(action port.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		authUser, err := getAuthUser(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
			c.Abort()
			return
		}
		allowed := s.authz.Enforce(authUser, action, "")
		if !allowed {
			apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
			c.JSON(apiErr.GetHttpErrorCode(), apiErr)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *PspHandler) Create(c *gin.Context) {
	var input CreateGatewayRequest
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

	s.logger.Debug("creating PSP", "input", input)
	psp, err := s.gatewayService.CreateGateway(c.Request.Context(), port.CreateGatewayInput{
		OrgId:    authUser.OrgId,
		PspId:    domain.Gateway(input.PspId),
		Name:     input.Name,
		Settings: input.Settings,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, NewGatewayFromEntity(psp))
}
