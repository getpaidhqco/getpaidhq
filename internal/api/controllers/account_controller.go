package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/domain/accounts"
	"payloop/internal/lib"
	"payloop/internal/services"
)

// AccountController data type
type AccountController struct {
	service services.AccountService
	logger  lib.Logger
}

// NewAccountController creates new user controller
func NewAccountController(service services.AccountService, logger lib.Logger) AccountController {
	return AccountController{
		service: service,
		logger:  logger,
	}
}

func (u AccountController) Create(c *gin.Context) {
	var input accounts.CreateAccountInput

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
