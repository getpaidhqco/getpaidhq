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
	fuego.Post(g, "", u.Create, option.Summary("Create an organization"))
}

func (u *OrgHandler) Create(c fuego.ContextWithBody[CreateOrgInput]) (any, error) {
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return nil, err
	}
	t, err := u.service.Create(c.Context(), port.CreateOrgInput{
		Owner:    authUser,
		Name:     input.Name,
		Country:  input.Country,
		Timezone: input.Timezone,
		Metadata: input.Metadata,
	})
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	return t, nil
}
