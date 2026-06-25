package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type InvoiceSettingsHandler struct {
	service *service.InvoiceSettingsService
	logger  port.Logger
	authz   port.Authz
}

func NewInvoiceSettingsHandler(s *service.InvoiceSettingsService, logger port.Logger, authz port.Authz) *InvoiceSettingsHandler {
	return &InvoiceSettingsHandler{service: s, logger: logger, authz: authz}
}

// InvoiceSettingsDTO is the wire shape for the per-tenant invoice reference
// policy: the prefix and zero-padding width used to render a human invoice
// reference (e.g. "INV-", 6 → "INV-000042").
type InvoiceSettingsDTO struct {
	Prefix  string `json:"prefix"`
	Padding int    `json:"padding" validate:"gte=0"`
}

func (h *InvoiceSettingsHandler) RegisterRoutes(s *fuego.Server) {
	g := fuego.Group(s, "/billing/invoice-settings", option.Tags("Billing"))
	fuego.Get(g, "", h.Get, option.Summary("Get invoice settings"), option.OperationID("getInvoiceSettings"))
	fuego.Put(g, "", h.Put, option.Summary("Set invoice settings"), option.OperationID("updateInvoiceSettings"))
}

func (h *InvoiceSettingsHandler) Get(c fuego.ContextNoBody) (InvoiceSettingsDTO, error) {
	if err := enforce(c, h.authz, port.ActionGetSetting); err != nil {
		return InvoiceSettingsDTO{}, err
	}
	authUser := AuthUserFrom(c)
	cfg, err := h.service.ResolveInvoiceSettings(c.Context(), authUser.OrgId)
	if err != nil {
		return InvoiceSettingsDTO{}, NewApiErrorFromError(err)
	}
	return toInvoiceSettingsDTO(cfg), nil
}

func (h *InvoiceSettingsHandler) Put(c fuego.ContextWithBody[InvoiceSettingsDTO]) (InvoiceSettingsDTO, error) {
	if err := enforce(c, h.authz, port.ActionUpdateSetting); err != nil {
		return InvoiceSettingsDTO{}, err
	}
	authUser := AuthUserFrom(c)
	body, err := c.Body()
	if err != nil {
		return InvoiceSettingsDTO{}, err
	}
	cfg := fromInvoiceSettingsDTO(body)
	if err := h.service.SetInvoiceSettings(c.Context(), authUser.OrgId, cfg); err != nil {
		return InvoiceSettingsDTO{}, NewApiErrorFromError(err)
	}
	return toInvoiceSettingsDTO(cfg), nil
}

func toInvoiceSettingsDTO(cfg domain.InvoiceSettings) InvoiceSettingsDTO {
	return InvoiceSettingsDTO{Prefix: cfg.Prefix, Padding: cfg.Padding}
}

func fromInvoiceSettingsDTO(dto InvoiceSettingsDTO) domain.InvoiceSettings {
	return domain.InvoiceSettings{Prefix: dto.Prefix, Padding: dto.Padding}
}
