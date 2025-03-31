package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
)

type PspController struct {
	gatewayService interfaces.GatewayService
	logger         logger.Logger
}

func NewPspController(gatewayService interfaces.GatewayService, logger logger.Logger) PspController {
	return PspController{
		gatewayService: gatewayService,
		logger:         logger,
	}
}

func (s PspController) Create(c *gin.Context) {
	var input request.CreateGatewayRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	s.logger.Debug("Creating PSP", "input", input)
	psp, err := s.gatewayService.CreateGateway(c.Request.Context(), dto.CreateGatewayInput{
		OrgId:    authUser.OrgId,
		PspId:    common.Gateway(input.PspId),
		Name:     input.Name,
		Settings: input.Settings,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewGatewayFromEntity(psp))
}
