// Package limenauth bridges the limen organization plugin to the app's
// authorization (Cedar) and eventing (NATS) infrastructure.
//
// The plugin ships no policy of its own: it gathers facts (actor, target
// organization, membership) and delegates every privileged action to the
// AuthorizeFunc built here, which evaluates the existing Cedar policy set.
package limenauth

import (
	"context"
	"fmt"

	organization "github.com/thecodearcher/limen/plugins/organization"

	"getpaidhq/internal/core/port"
)

// NewOrganizationAuthorizer returns the AuthorizeFunc consulted by the limen
// organization plugin for every privileged action. It derives a role from the
// facts the plugin provides and evaluates it against the Cedar policy set,
// reusing the app's Role×Action model (see policy.cedar).
func NewOrganizationAuthorizer(authz port.Authz, logger port.Logger) organization.AuthorizeFunc {
	return func(_ context.Context, req organization.AuthzRequest) error {
		// Onboarding: creating an organization has no target org or membership
		// yet, mirroring the POST /api/organizations bypass in the authn
		// middleware. Any authenticated user may create one.
		if req.Action == organization.ActionOrganizationCreate {
			return nil
		}

		authUser := port.AuthUser{
			OrgId:       fmt.Sprint(req.Organization.ID),
			Id:          fmt.Sprint(req.User.ID),
			Email:       req.User.Email,
			PrimaryRole: roleFromFacts(req),
		}

		if !authz.Enforce(authUser, port.Action(req.Action), "organization") {
			logger.Warnf("limen authz denied: action=%s user=%s org=%v role=%s",
				req.Action, req.User.Email, req.Organization.ID, authUser.PrimaryRole)
			return organization.ErrForbidden
		}
		return nil
	}
}

// roleFromFacts maps the plugin's membership facts onto the app's role model:
// the organization creator is its owner, any other member is a member, and
// non-members carry a role Cedar has no permit rules for (deny by default).
//
// Roles are derived rather than stored because the limen plugin keeps
// membership as a pure fact — when the app grows real per-membership roles
// they should be looked up here instead.
func roleFromFacts(req organization.AuthzRequest) port.UserRole {
	if req.Membership == nil {
		return port.UserRole("none")
	}
	if req.Organization != nil && req.Organization.CreatedBy != nil &&
		fmt.Sprint(req.Organization.CreatedBy) == fmt.Sprint(req.User.ID) {
		return port.RoleOwner
	}
	return port.RoleMember
}
