package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// SettingHandler is CRUD over org-scoped key/value settings, keyed by (parent_id, id).
type SettingHandler struct {
	settingService *service.SettingService
	logger         port.Logger
	authz          port.Authz
}

func NewSettingHandler(settingService *service.SettingService, logger port.Logger, authz port.Authz) *SettingHandler {
	return &SettingHandler{settingService: settingService, logger: logger, authz: authz}
}

func (h *SettingHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/settings", option.Tags("Settings"))
	fuego.Post(g, "", h.Create, option.Summary("Create a setting"))
	fuego.Get(g, "", h.List, append(PaginationParams(), option.Summary("List settings (optional ?parent_id=)"))...)
	fuego.Get(g, "/{parentId}/{id}", h.Get, option.Summary("Get a setting"))
	fuego.Put(g, "/{parentId}/{id}", h.Update, option.Summary("Create or replace a setting"))
	fuego.Delete(g, "/{parentId}/{id}", h.Delete, option.Summary("Delete a setting"))
}

func (h *SettingHandler) Create(c fuego.ContextWithBody[CreateSettingRequest]) (SettingResponse, error) {
	if err := enforce(c, h.authz, port.ActionCreateSetting); err != nil {
		return SettingResponse{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return SettingResponse{}, err
	}
	setting, err := h.settingService.Create(c.Context(), service.CreateSettingInput{
		OrgId: authUser.OrgId, ParentId: req.ParentId, Id: req.Id, Type: req.Type, Value: req.Value,
	})
	if err != nil {
		return SettingResponse{}, NewApiErrorFromError(err)
	}
	return NewSettingResponse(setting), nil
}

func (h *SettingHandler) Get(c fuego.ContextNoBody) (SettingResponse, error) {
	if err := enforce(c, h.authz, port.ActionGetSetting); err != nil {
		return SettingResponse{}, err
	}
	authUser := AuthUserFrom(c)
	setting, err := h.settingService.Get(c.Context(), authUser.OrgId, c.PathParam("parentId"), c.PathParam("id"))
	if err != nil {
		return SettingResponse{}, NewApiErrorFromError(err)
	}
	return NewSettingResponse(setting), nil
}

func (h *SettingHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, h.authz, port.ActionListSettings); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	p := GetPagination(c)
	settings, total, err := h.settingService.List(c.Context(), authUser.OrgId, c.QueryParam("parent_id"), p)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	out := make([]SettingResponse, len(settings))
	for i, s := range settings {
		out[i] = NewSettingResponse(s)
	}
	return ListResponse{Data: out, Meta: Meta{Total: total, Page: p.Page, Limit: p.Limit}}, nil
}

func (h *SettingHandler) Update(c fuego.ContextWithBody[UpdateSettingRequest]) (SettingResponse, error) {
	if err := enforce(c, h.authz, port.ActionUpdateSetting); err != nil {
		return SettingResponse{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return SettingResponse{}, err
	}
	setting, err := h.settingService.Upsert(c.Context(), service.CreateSettingInput{
		OrgId: authUser.OrgId, ParentId: c.PathParam("parentId"), Id: c.PathParam("id"), Type: req.Type, Value: req.Value,
	})
	if err != nil {
		return SettingResponse{}, NewApiErrorFromError(err)
	}
	return NewSettingResponse(setting), nil
}

func (h *SettingHandler) Delete(c fuego.ContextNoBody) (EmptyResponse, error) {
	if err := enforce(c, h.authz, port.ActionDeleteSetting); err != nil {
		return EmptyResponse{}, err
	}
	authUser := AuthUserFrom(c)
	if err := h.settingService.Delete(c.Context(), authUser.OrgId, c.PathParam("parentId"), c.PathParam("id")); err != nil {
		return EmptyResponse{}, NewApiErrorFromError(err)
	}
	c.SetStatus(204)
	return EmptyResponse{}, nil
}
