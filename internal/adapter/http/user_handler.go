package handler

import (
	"github.com/go-fuego/fuego"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type UserHandler struct {
	service *service.UserService
	logger  port.Logger
}

func NewUserHandler(userService *service.UserService, logger port.Logger) *UserHandler {
	return &UserHandler{service: userService, logger: logger}
}

// RegisterRoutes is a placeholder. No user routes are wired today; the
// type exists so the application can hold a UserService for future use.
func (u *UserHandler) RegisterRoutes(_ *fuego.Server) {}
