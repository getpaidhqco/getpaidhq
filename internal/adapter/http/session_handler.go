package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type SessionHandler struct {
	sessionService *service.SessionService
	logger         port.Logger
	authz          port.Authz
}

func NewSessionHandler(sessionService *service.SessionService, logger port.Logger, authz port.Authz) *SessionHandler {
	return &SessionHandler{sessionService: sessionService, logger: logger, authz: authz}
}

func (s *SessionHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/sessions", option.Tags("Sessions"))
	fuego.Post(g, "", s.Create, option.Summary("Create a session"))
}

func (s *SessionHandler) Create(c fuego.ContextWithBody[CreateSessionRequest]) (CreateSessionResponse, error) {
	if err := enforce(c, s.authz, port.ActionCreateSession); err != nil {
		return CreateSessionResponse{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return CreateSessionResponse{}, err
	}
	s.logger.Debug("Creating session", "input", req)
	session, err := s.sessionService.CreateSession(c.Context(), req.ToInput(authUser.OrgId))
	if err != nil {
		return CreateSessionResponse{}, NewApiErrorFromError(err)
	}
	return NewCreateSessionResponse(session), nil
}
