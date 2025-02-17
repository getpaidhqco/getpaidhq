package lib

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/application/lib/logger"
)

// RequestHandler function
type RequestHandler struct {
	Gin *gin.Engine
}

// NewRequestHandler creates a new request handler
func NewRequestHandler(logger logger.Logger) RequestHandler {
	engine := gin.Default()
	return RequestHandler{Gin: engine}
}
