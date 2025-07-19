package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/mappers"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
)

// OrgController data type
type OrgController struct {
	service services.OrgService
	logger  logger.Logger
}

// NewOrgController creates new user controller
func NewOrgController(service services.OrgService, logger logger.Logger) OrgController {
	return OrgController{
		service: service,
		logger:  logger,
	}
}

func (u OrgController) GetApiKeys(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId

	apiKeys, err := u.service.GetApiKeys(c.Request.Context(), orgId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	response := mappers.ToApiKeyListResponse(apiKeys)
	c.JSON(200, response)
}

func (u OrgController) Create(c *gin.Context) {
	var input request.CreateOrgInput
	user, _ := c.Get("user")
	authUser := user.(authn.User)

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	t, err := u.service.Create(c.Request.Context(), dto.CreateOrgInput{
		Owner:    authUser,
		Name:     input.Name,
		Country:  input.Country,
		Timezone: input.Timezone,
		Metadata: input.Metadata,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, t)
}
