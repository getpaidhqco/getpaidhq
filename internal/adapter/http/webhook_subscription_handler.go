package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type WebhookSubscriptionHandler struct {
	webhookSubscriptionService *service.WebhookSubscriptionService
	logger                     port.Logger
	authz                      port.Authz
}

func NewWebhookSubscriptionHandler(
	webhookSubscriptionService *service.WebhookSubscriptionService,
	logger port.Logger,
	authz port.Authz,
) *WebhookSubscriptionHandler {
	return &WebhookSubscriptionHandler{
		webhookSubscriptionService: webhookSubscriptionService,
		logger:                     logger,
		authz:                      authz,
	}
}

func (s *WebhookSubscriptionHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/webhooks", option.Tags("Webhook Subscriptions"))
	fuego.Post(g, "", s.Create, option.Summary("Create a webhook subscription"))
	fuego.Get(g, "", s.List, option.Summary("List webhook subscriptions"))
}

func (s *WebhookSubscriptionHandler) Create(c fuego.ContextWithBody[CreateWebhookSubscriptionRequest]) (any, error) {
	if err := enforce(c, s.authz, port.ActionCreateWebhookSubscription); err != nil {
		return nil, err
	}
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return nil, err
	}
	webhook, err := s.webhookSubscriptionService.Create(c.Context(), service.CreateWebhookSubscriptionInput{
		OrgId:  authUser.OrgId,
		Url:    input.Url,
		Events: input.Events,
		Secret: input.Secret,
	})
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	return webhook, nil
}

// List is a placeholder — the gin code mistakenly pointed GET /webhooks
// at the Create handler. The service does not have a List method yet, so
// the route now returns an empty list until that surface lands. The
// route is kept registered to preserve API surface compatibility.
func (s *WebhookSubscriptionHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, s.authz, port.ActionListWebhookSubscriptions); err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Data: []any{}, Meta: Meta{}}, nil
}
