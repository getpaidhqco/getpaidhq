package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type PspHandler struct {
	gatewayService *service.PspService
	logger         port.Logger
	authz          port.Authz
}

func NewPspHandler(gatewayService *service.PspService, logger port.Logger, authz port.Authz) *PspHandler {
	return &PspHandler{gatewayService: gatewayService, logger: logger, authz: authz}
}

func (s *PspHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/gateways", option.Tags("Payment Service Providers"))
	fuego.Post(g, "", s.Create, option.Summary("Configure a payment service provider"))
}

func (s *PspHandler) Create(c fuego.ContextWithBody[CreateGatewayRequest]) (GatewayResponse, error) {
	if err := enforce(c, s.authz, port.ActionCreatePaymentServiceProvider); err != nil {
		return GatewayResponse{}, err
	}
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return GatewayResponse{}, err
	}
	s.logger.Debug("Creating PSP", "input", input)
	psp, err := s.gatewayService.CreateGateway(c.Context(), port.CreateGatewayInput{
		OrgId:    authUser.OrgId,
		PspId:    domain.Gateway(input.PspId),
		Name:     input.Name,
		Settings: input.Settings,
	})
	if err != nil {
		return GatewayResponse{}, NewApiErrorFromError(err)
	}
	return NewGatewayFromEntity(psp), nil
}
