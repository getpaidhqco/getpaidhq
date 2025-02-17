package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	services2 "payloop/internal/application/services"
	"payloop/internal/domain/entities/sessions"
)

type SessionController struct {
	sessionService interfaces.SessionService
	cartService    services2.CartService
	logger         logger.Logger
}

func NewSessionController(sessionService interfaces.SessionService, cartService services2.CartService, logger logger.Logger) SessionController {
	return SessionController{
		sessionService: sessionService,
		cartService:    cartService,
		logger:         logger,
	}
}

func (s SessionController) Create(c *gin.Context) {
	var input sessions.CreateSessionRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	s.logger.Debug("Creating session", "input", input)
	session, err := s.sessionService.CreateSession(c.Request.Context(), sessions.CreateSessionInput{
		OrgId:    authUser.OrgId,
		Currency: input.Currency,
		Country:  input.Country,
		Metadata: nil,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, sessions.CreateSessionResponse{
		Id:     session.Id,
		CartId: session.CartId,
	})
}
