package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type OrgHandler struct {
	service *service.OrgService
	logger  port.Logger
}

func NewOrgHandler(service *service.OrgService, logger port.Logger) *OrgHandler {
	return &OrgHandler{service: service, logger: logger}
}

func (u *OrgHandler) RegisterRoutes(s *fuego.Server) {
	g := fuego.Group(s, "/organizations", option.Tags("Organizations"))
	fuego.Post(g, "", u.Create, option.Summary("Create an organization"), option.OperationID("createOrganization"))
}

func (u *OrgHandler) Create(c fuego.ContextWithBody[CreateOrgRequest]) (OrgResponse, error) {
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return OrgResponse{}, err
	}
	org, err := u.service.Create(c.Context(), req.ToInput(authUser))
	if err != nil {
		return OrgResponse{}, NewApiErrorFromError(err)
	}
	return NewOrgResponse(org), nil
}
