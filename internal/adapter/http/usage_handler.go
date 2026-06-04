package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type UsageHandler struct {
	usageService *service.UsageService
	logger       port.Logger
}

func NewUsageHandler(usageService *service.UsageService, logger port.Logger) *UsageHandler {
	return &UsageHandler{usageService: usageService, logger: logger}
}

func (h *UsageHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/usage", option.Tags("Usage"))
	fuego.Post(g, "/events", h.RecordEvent, option.Summary("Record a usage event"))
}

func (h *UsageHandler) RecordEvent(c fuego.ContextWithBody[RecordEventRequest]) (RecordEventResponse, error) {
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return RecordEventResponse{}, err
	}
	res, err := h.usageService.RecordEvent(c.Context(), req.ToInput(authUser.OrgId))
	if err != nil {
		return RecordEventResponse{}, NewApiErrorFromError(err)
	}
	return NewRecordEventResponse(res), nil
}
