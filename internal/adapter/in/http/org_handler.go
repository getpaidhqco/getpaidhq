package handler

import (
	"github.com/gin-gonic/gin"

	"payloop/internal/application/dto"
	"payloop/internal/application/services"
	"payloop/internal/core/port"
)

// OrgHandler handles HTTP requests for organizations.
type OrgHandler struct {
	service services.OrgService
	logger  port.Logger
}

// NewOrgHandler creates a new OrgHandler.
func NewOrgHandler(service services.OrgService, logger port.Logger) *OrgHandler {
	return &OrgHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers organization routes on the given router group.
func (u *OrgHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/organizations", u.Create)
}

func (u *OrgHandler) Create(c *gin.Context) {
	var input CreateOrgInput
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
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
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, t)
}
