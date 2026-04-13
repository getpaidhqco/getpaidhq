package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

// PspHandler handles HTTP requests for payment service providers.
type PspHandler struct {
	gatewayService interfaces.GatewayService
	logger         port.Logger
	authz          port.Authz
}

// NewPspHandler creates a new PspHandler.
func NewPspHandler(gatewayService interfaces.GatewayService, logger port.Logger, authz port.Authz) *PspHandler {
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
		user, _ := c.Get("user")
		authUser := user.(port.AuthUser)
		allowed := s.authz.Enforce(authUser, action, "")
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

func (s *PspHandler) Create(c *gin.Context) {
	var input CreateGatewayRequest
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	s.logger.Debug("Creating PSP", "input", input)
	psp, err := s.gatewayService.CreateGateway(c.Request.Context(), dto.CreateGatewayInput{
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
