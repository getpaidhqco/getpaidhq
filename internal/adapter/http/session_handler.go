package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
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

func (s *SessionHandler) Create(c fuego.ContextWithBody[domain.CreateSessionRequest]) (domain.CreateSessionResponse, error) {
	if err := enforce(c, s.authz, port.ActionCreateSession); err != nil {
		return domain.CreateSessionResponse{}, err
	}
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return domain.CreateSessionResponse{}, err
	}
	s.logger.Debug("Creating session", "input", input)
	session, err := s.sessionService.CreateSession(c.Context(), service.CreateSessionInput{
		OrgId:    authUser.OrgId,
		Currency: input.Currency,
		Country:  input.Country,
		Metadata: nil,
	})
	if err != nil {
		return domain.CreateSessionResponse{}, NewApiErrorFromError(err)
	}
	return domain.CreateSessionResponse{
		Id:     session.Id,
		CartId: session.CartId,
	}, nil
}
