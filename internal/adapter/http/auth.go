package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/port"
)

// getAuthUser safely extracts the authenticated user from the gin context.
// It returns an error if the user is not set or has an unexpected type.
func getAuthUser(c *gin.Context) (port.AuthUser, error) {
	user, exists := c.Get("user")
	if !exists {
		return port.AuthUser{}, errors.New("authentication required")
	}
	authUser, ok := user.(port.AuthUser)
	if !ok {
		return port.AuthUser{}, errors.New("invalid authentication context")
	}
	return authUser, nil
}
