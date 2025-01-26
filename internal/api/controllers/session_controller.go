package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/domain/sessions"
	"payloop/internal/lib"
	"payloop/internal/services"
)

type SessionController struct {
	sessionService services.SessionService
	cartService    services.CartService
	logger         lib.Logger
}

func NewSessionController(sessionService services.SessionService, cartService services.CartService, logger lib.Logger) SessionController {
	return SessionController{
		sessionService: sessionService,
		cartService:    cartService,
		logger:         logger,
	}
}

func (s SessionController) Create(c *gin.Context) {
	var input sessions.CreateSessionRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	s.logger.Debug("Creating session", "input", input)

	session, err := s.sessionService.CreateSession(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, sessions.CreateSessionResponse{
		Id:     session.Id,
		CartId: session.CartId,
	})
}
