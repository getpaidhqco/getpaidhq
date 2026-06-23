package handler

import (
	"time"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type ReminderConfigHandler struct {
	service *service.ReminderConfigService
	logger  port.Logger
}

func NewReminderConfigHandler(s *service.ReminderConfigService, logger port.Logger) *ReminderConfigHandler {
	return &ReminderConfigHandler{service: s, logger: logger}
}

// ReminderConfigDTO is the wire shape for the per-tenant renewal-reminder
// policy. Offsets are human-readable durations (e.g. "168h", "24h").
type ReminderConfigDTO struct {
	Enabled bool     `json:"enabled"`
	Offsets []string `json:"offsets" validate:"dive,required"`
}

func (h *ReminderConfigHandler) RegisterRoutes(s *fuego.Server) {
	g := fuego.Group(s, "/billing/reminder-config", option.Tags("Billing"))
	fuego.Get(g, "", h.Get, option.Summary("Get renewal reminder config"), option.OperationID("getReminderConfig"))
	fuego.Put(g, "", h.Put, option.Summary("Set renewal reminder config"), option.OperationID("updateReminderConfig"))
}

func (h *ReminderConfigHandler) Get(c fuego.ContextNoBody) (ReminderConfigDTO, error) {
	authUser := AuthUserFrom(c)
	cfg, err := h.service.ResolveReminderConfig(c.Context(), authUser.OrgId)
	if err != nil {
		return ReminderConfigDTO{}, NewApiErrorFromError(err)
	}
	return toReminderDTO(cfg), nil
}

func (h *ReminderConfigHandler) Put(c fuego.ContextWithBody[ReminderConfigDTO]) (ReminderConfigDTO, error) {
	authUser := AuthUserFrom(c)
	body, err := c.Body()
	if err != nil {
		return ReminderConfigDTO{}, err
	}
	cfg, err := fromReminderDTO(body)
	if err != nil {
		return ReminderConfigDTO{}, fuego.BadRequestError{Title: "invalid offset duration", Detail: err.Error()}
	}
	if err := h.service.SetReminderConfig(c.Context(), authUser.OrgId, cfg); err != nil {
		return ReminderConfigDTO{}, NewApiErrorFromError(err)
	}
	return toReminderDTO(cfg), nil
}

func toReminderDTO(cfg domain.ReminderConfig) ReminderConfigDTO {
	dto := ReminderConfigDTO{Enabled: cfg.Enabled}
	for _, d := range cfg.Offsets {
		dto.Offsets = append(dto.Offsets, d.String())
	}
	return dto
}

func fromReminderDTO(dto ReminderConfigDTO) (domain.ReminderConfig, error) {
	cfg := domain.ReminderConfig{Enabled: dto.Enabled}
	for _, s := range dto.Offsets {
		d, err := time.ParseDuration(s)
		if err != nil {
			return domain.ReminderConfig{}, err
		}
		cfg.Offsets = append(cfg.Offsets, d)
	}
	return cfg, nil
}
