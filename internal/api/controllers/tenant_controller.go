package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/lib"
	"payloop/internal/repository/tenants"
	"payloop/internal/services"
)

// TenantController data type
type TenantController struct {
	service services.TenantService
	logger  lib.Logger
}

// NewTenantController creates new user controller
func NewTenantController(tenantService services.TenantService, logger lib.Logger) TenantController {
	return TenantController{
		service: tenantService,
		logger:  logger,
	}
}

func (u TenantController) Create(c *gin.Context) {
	var input tenants.CreateTenantInput

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
