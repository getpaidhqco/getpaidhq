package controllers

import (
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
)

// UserController data type
type UserController struct {
	service services.UserService
	logger  logger.Logger
}

// NewUserController creates new user controller
func NewUserController(userService services.UserService, logger logger.Logger) UserController {
	return UserController{
		service: userService,
		logger:  logger,
	}
}
