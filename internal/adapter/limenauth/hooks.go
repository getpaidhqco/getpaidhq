package limenauth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/thecodearcher/limen"
	organization "github.com/thecodearcher/limen/plugins/organization"

	"getpaidhq/internal/core/port"
)

// orgEventTopics maps organization route IDs (part of the plugin's public
// contract) to the NATS topics the rest of the system consumes.
var orgEventTopics = map[limen.RouteID]string{
	organization.RouteIDCreate:            "limen.organization.created",
	organization.RouteIDUpdate:            "limen.organization.updated",
	organization.RouteIDDelete:            "limen.organization.deleted",
	organization.RouteIDMembersRemove:     "limen.organization.member.removed",
	organization.RouteIDLeave:             "limen.organization.member.left",
	organization.RouteIDInvitationsCreate: "limen.organization.invitation.created",
	organization.RouteIDInvitationsAccept: "limen.organization.invitation.accepted",
}

// EventHooks returns limen after-hooks that publish organization lifecycle
// events. The plugin deliberately has no event system; the hook layer is the
// app's chosen emission point, keyed on the plugin's exported route IDs.
//
// Publishing is fire-and-forget on a goroutine: after-hooks run in the request
// path before the response is flushed, so a slow NATS round-trip must not
// delay the client. Deliveries that need transactional guarantees should go
// through the outbox instead.
func EventHooks(publisher port.PubSub, logger port.Logger) *limen.Hooks {
	return &limen.Hooks{
		After: []*limen.Hook{
			{
				PathMatcher: func(ctx *limen.HookContext) bool {
					_, ok := orgEventTopics[limen.RouteID(ctx.RouteID())]
					return ok
				},
				Run: func(ctx *limen.HookContext) bool {
					response := ctx.GetResponse()
					if response == nil || response.IsError || response.StatusCode != http.StatusOK {
						return true
					}

					topic := orgEventTopics[limen.RouteID(ctx.RouteID())]
					payload := response.Payload
					go func() {
						if err := publisher.Publish(context.Background(), orgIDFromPayload(payload), topic, payload); err != nil {
							logger.Warnf("limen event publish failed: topic=%s err=%v", topic, err)
						}
					}()
					return true
				},
			},
		},
	}
}

// orgIDFromPayload extracts the organization ID from a serialized response
// payload when present; message-only responses (e.g. "member removed") have
// none.
func orgIDFromPayload(payload any) string {
	body, ok := payload.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"organization_id", "id"} {
		if value, exists := body[key]; exists && value != nil {
			return fmt.Sprint(value)
		}
	}
	return ""
}
