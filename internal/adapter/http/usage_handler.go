package handler

import (
	"errors"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

type UsageHandler struct {
	usageService *service.UsageService
	logger       port.Logger
	authz        port.Authz
}

func NewUsageHandler(usageService *service.UsageService, logger port.Logger, authz port.Authz) *UsageHandler {
	return &UsageHandler{usageService: usageService, logger: logger, authz: authz}
}

func (h *UsageHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/usage", option.Tags("Usage"))
	fuego.Post(g, "/ingest", h.Ingest, option.Summary("Ingest one or more usage events"), option.OperationID("ingestUsageEvents"))

	// Read current-period usage lives under the subscription it belongs to.
	subs := fuego.Group(srv, "/subscriptions", option.Tags("Usage"))
	fuego.Get(subs, "/{id}/usage", h.SubscriptionUsage, option.Summary("Get a subscription's current-period usage"), option.OperationID("getSubscriptionUsage"))
}

// Ingest records a batch of usage events. Each event is validated and stored
// independently: the response carries a per-event result and the request only
// fails as a whole on a malformed body, an empty batch, or an oversize batch.
func (h *UsageHandler) Ingest(c fuego.ContextWithBody[IngestEventsRequest]) (IngestEventsResponse, error) {
	if err := enforce(c, h.authz, port.ActionRecordUsage); err != nil {
		return IngestEventsResponse{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return IngestEventsResponse{}, err
	}
	results, err := h.usageService.RecordEvents(c.Context(), req.ToInputs(authUser.OrgId))
	if err != nil {
		return IngestEventsResponse{}, NewApiErrorFromError(err)
	}
	return NewIngestEventsResponse(results), nil
}

// SubscriptionUsage returns the subscription's metered usage for its current
// billing period.
func (h *UsageHandler) SubscriptionUsage(c fuego.ContextNoBody) (SubscriptionUsageResponse, error) {
	if err := enforce(c, h.authz, port.ActionReadUsage); err != nil {
		return SubscriptionUsageResponse{}, err
	}
	authUser := AuthUserFrom(c)
	usage, err := h.usageService.CurrentPeriodUsage(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		// Repos return port.ErrNotFound (distinct from lib.ErrNotFound); map it to 404.
		if errors.Is(err, port.ErrNotFound) {
			return SubscriptionUsageResponse{}, NewApiError(lib.NotFoundError, "subscription not found", err)
		}
		return SubscriptionUsageResponse{}, NewApiErrorFromError(err)
	}
	return NewSubscriptionUsageResponse(usage), nil
}
