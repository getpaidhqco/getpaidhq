package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// MeterHandler exposes BillableMetric (meter) management.
type MeterHandler struct {
	meterService *service.MeterService
	logger       port.Logger
	authz        port.Authz
}

func NewMeterHandler(meterService *service.MeterService, logger port.Logger, authz port.Authz) *MeterHandler {
	return &MeterHandler{meterService: meterService, logger: logger, authz: authz}
}

func (h *MeterHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/meters", option.Tags("Meters"))
	fuego.Post(g, "", h.Create, option.Summary("Create a meter"))
	fuego.Get(g, "", h.List, option.Summary("List meters"))
	fuego.Get(g, "/{id}", h.Get, option.Summary("Get a meter"))
}

func (h *MeterHandler) Create(c fuego.ContextWithBody[CreateMeterRequest]) (MeterResponse, error) {
	if err := enforce(c, h.authz, port.ActionCreateMeter); err != nil {
		return MeterResponse{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return MeterResponse{}, err
	}
	metric, err := h.meterService.Create(c.Context(), req.ToInput(authUser.OrgId))
	if err != nil {
		return MeterResponse{}, NewApiErrorFromError(err)
	}
	return NewMeterResponse(metric), nil
}

func (h *MeterHandler) Get(c fuego.ContextNoBody) (MeterResponse, error) {
	if err := enforce(c, h.authz, port.ActionGetMeter); err != nil {
		return MeterResponse{}, err
	}
	authUser := AuthUserFrom(c)
	metric, err := h.meterService.Get(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return MeterResponse{}, NewApiErrorFromError(err)
	}
	return NewMeterResponse(metric), nil
}

func (h *MeterHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, h.authz, port.ActionListMeters); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	pagination := GetPagination(c)
	metrics, total, err := h.meterService.List(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	out := make([]MeterResponse, len(metrics))
	for i, m := range metrics {
		out[i] = NewMeterResponse(m)
	}
	return ListResponse{
		Data: out,
		Meta: Meta{Total: total, Page: pagination.Page, Limit: pagination.Limit},
	}, nil
}
