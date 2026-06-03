package handler

import (
	"time"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// ApiKeyHandler exposes CRUD for org-scoped API keys. The plaintext key
// is returned exactly once from Create; List/Get never expose it.
type ApiKeyHandler struct {
	apiKeyService *service.ApiKeyService
	logger        port.Logger
	authz         port.Authz
}

func NewApiKeyHandler(apiKeyService *service.ApiKeyService, logger port.Logger, authz port.Authz) *ApiKeyHandler {
	return &ApiKeyHandler{apiKeyService: apiKeyService, logger: logger, authz: authz}
}

func (h *ApiKeyHandler) RegisterRoutes(s *fuego.Server) {
	g := fuego.Group(s, "/api-keys", option.Tags("API Keys"))
	fuego.Get(g, "", h.List, option.Summary("List API keys"))
	fuego.Post(g, "", h.Create, option.Summary("Create an API key"))
	fuego.Delete(g, "/{id}", h.Delete, option.Summary("Revoke an API key"))
}

// CreateApiKeyInput is the request body for POST /api/api-keys.
type CreateApiKeyInput struct {
	// Name is an optional human-readable label for the key (e.g.
	// "ci-deploy"). 64 chars max. Empty is fine.
	Name string `json:"name" validate:"omitempty,max=64"`
}

// ApiKeyResponse is the safe-to-leak shape (no secret, no hash).
type ApiKeyResponse struct {
	Id        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ApiKeyCreateResponse adds the plaintext key, returned ONCE at creation.
// Clients must store it now — there is no recovery flow.
type ApiKeyCreateResponse struct {
	ApiKeyResponse
	// Key is the plaintext token (sk_<id>_<secret>). Shown ONCE; never
	// stored server-side and never retrievable afterwards.
	Key string `json:"key"`
}

func (h *ApiKeyHandler) Create(c fuego.ContextWithBody[CreateApiKeyInput]) (ApiKeyCreateResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionCreateApiKey, "") {
		return ApiKeyCreateResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return ApiKeyCreateResponse{}, err
	}

	k, err := h.apiKeyService.Create(c.Context(), authUser.OrgId, input.Name)
	if err != nil {
		return ApiKeyCreateResponse{}, NewApiErrorFromError(err)
	}
	c.SetStatus(201)
	return ApiKeyCreateResponse{
		ApiKeyResponse: ApiKeyResponse{
			Id:        k.Id,
			Name:      k.Name,
			CreatedAt: k.CreatedAt,
			UpdatedAt: k.UpdatedAt,
		},
		Key: k.RawKey,
	}, nil
}

func (h *ApiKeyHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionListApiKeys, "") {
		return ListResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	pagination := GetPagination(c)

	keys, total, err := h.apiKeyService.List(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	out := make([]ApiKeyResponse, len(keys))
	for i, k := range keys {
		out[i] = ApiKeyResponse{
			Id:        k.Id,
			Name:      k.Name,
			CreatedAt: k.CreatedAt,
			UpdatedAt: k.UpdatedAt,
		}
	}
	return ListResponse{
		Data: out,
		Meta: Meta{Total: total, Page: pagination.Page, Limit: pagination.Limit},
	}, nil
}

func (h *ApiKeyHandler) Delete(c fuego.ContextNoBody) (EmptyResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionDeleteApiKey, "") {
		return EmptyResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	if err := h.apiKeyService.Delete(c.Context(), authUser.OrgId, c.PathParam("id")); err != nil {
		return EmptyResponse{}, NewApiErrorFromError(err)
	}
	c.SetStatus(204)
	return EmptyResponse{}, nil
}
