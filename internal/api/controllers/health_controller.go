package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/lib"
)

// HealthController data type
type HealthController struct {
	logger lib.Logger
}

func NewHealthController(logger lib.Logger) HealthController {
	return HealthController{
		logger: logger,
	}
}

func (u HealthController) Healthcheck(c *gin.Context) {
	c.JSON(200, map[string]string{"status": "ök"})
}
