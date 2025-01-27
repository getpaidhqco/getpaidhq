package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/domain/orgs"
	"payloop/internal/lib"
	"payloop/internal/services"
)

// OrgController data type
type OrgController struct {
	service services.OrgService
	logger  lib.Logger
}

// NewOrgController creates new user controller
func NewOrgController(service services.OrgService, logger lib.Logger) OrgController {
	return OrgController{
		service: service,
		logger:  logger,
	}
}

func (u OrgController) Create(c *gin.Context) {
	var input orgs.CreateOrgInput

	if err := c.ShouldBindJSON(&input); err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	u.logger.Debug("Creating tenant", "input", input)
	t, err := u.service.Create(c.Request.Context(), input)
	if err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, t)
}
