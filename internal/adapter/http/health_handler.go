package handler

import (
	"github.com/gin-gonic/gin"

	"getpaidhq/internal/core/port"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	logger port.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(logger port.Logger) *HealthHandler {
	return &HealthHandler{
		logger: logger,
	}
}

// RegisterRoutes registers health routes on the given router group.
func (u *HealthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", u.Healthcheck)
}

func (u *HealthHandler) Healthcheck(c *gin.Context) {
	c.JSON(200, map[string]string{"status": "ok"})
}
