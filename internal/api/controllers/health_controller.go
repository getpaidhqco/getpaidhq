package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/application/lib/logger"
)

// HealthController data type
type HealthController struct {
	logger logger.Logger
}

func NewHealthController(logger logger.Logger) HealthController {
	return HealthController{
		logger: logger,
	}
}

func (u HealthController) Healthcheck(c *gin.Context) {
	panic("simulated panic for testing")
	c.JSON(200, map[string]string{"status": "ök"})
}
