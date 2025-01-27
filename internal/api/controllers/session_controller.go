package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	services2 "payloop/internal/application/services"
	"payloop/internal/domain/entities/sessions"
	"payloop/internal/lib"
)

type SessionController struct {
	sessionService services2.SessionService
	cartService    services2.CartService
	logger         lib.Logger
}

func NewSessionController(sessionService services2.SessionService, cartService services2.CartService, logger lib.Logger) SessionController {
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
