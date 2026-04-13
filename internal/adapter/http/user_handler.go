package handler

import (
	"github.com/gin-gonic/gin"

	"payloop/internal/core/service"
	"payloop/internal/core/port"
)

// UserHandler handles HTTP requests for users.
type UserHandler struct {
	service service.UserService
	logger  port.Logger
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService service.UserService, logger port.Logger) *UserHandler {
	return &UserHandler{
		service: userService,
		logger:  logger,
	}
}

// RegisterRoutes registers user routes on the given router group.
// Currently no routes are defined for users.
func (u *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// No routes defined yet.
}
