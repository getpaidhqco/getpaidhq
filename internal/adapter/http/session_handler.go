package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/core/service"
	"payloop/internal/lib"
)

// SessionHandler handles HTTP requests for sessions.
type SessionHandler struct {
	sessionService *service.SessionService
	logger         port.Logger
	authz          port.Authz
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(sessionService *service.SessionService, logger port.Logger, authz port.Authz) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		logger:         logger,
		authz:          authz,
	}
}

// RegisterRoutes registers session routes on the given router group.
func (s *SessionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/sessions", s.checkAuthz(port.ActionCreateSession), s.Create)
}

func (s *SessionHandler) checkAuthz(action port.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		authUser, err := getAuthUser(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
			c.Abort()
			return
		}
		allowed := s.authz.Enforce(authUser, action, "")
		if !allowed {
			apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
			c.JSON(apiErr.GetHttpErrorCode(), apiErr)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *SessionHandler) Create(c *gin.Context) {
	var input domain.CreateSessionRequest
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	s.logger.Debug("creating session", "input", input)
	session, err := s.sessionService.CreateSession(c.Request.Context(), domain.CreateSessionInput{
		OrgId:    authUser.OrgId,
		Currency: input.Currency,
		Country:  input.Country,
		Metadata: nil,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, domain.CreateSessionResponse{
		Id:     session.Id,
		CartId: session.CartId,
	})
}
