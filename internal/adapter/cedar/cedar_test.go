package cedar

import (
	"os"
	"path/filepath"
	"testing"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"

	"github.com/cedar-policy/cedar-go"
	"github.com/cedar-policy/cedar-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopLogger satisfies lib.Logger (== port.Logger) without producing output.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any)  {}
func (noopLogger) Info(string, ...any)   {}
func (noopLogger) Warn(string, ...any)   {}
func (noopLogger) Error(string, ...any)  {}
func (noopLogger) Fatal(string, ...any)  {}
func (noopLogger) Debugf(string, ...any) {}
func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Warnf(string, ...any)  {}
func (noopLogger) Errorf(string, ...any) {}
func (noopLogger) Panicf(string, ...any) {}
func (noopLogger) Fatalf(string, ...any) {}
func (noopLogger) Sync() error           { return nil }

var _ lib.Logger = noopLogger{}

// repoRootPolicyPath walks up from the package dir (the CWD under `go test`)
// to find the real repo-root policy.cedar.
func repoRootPolicyPath(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		candidate := filepath.Join(dir, "policy.cedar")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "reached filesystem root without finding policy.cedar")
		dir = parent
	}
}

// newTestAuthz builds the REAL Cedar adapter against the REAL repo-root
// policy.cedar, exactly the way config.NewApp wires it.
func newTestAuthz(t *testing.T) port.Authz {
	t.Helper()
	return NewCedarAuthz(noopLogger{}, repoRootPolicyPath(t))
}

func user(role port.UserRole) port.AuthUser {
	return port.AuthUser{OrgId: "org-1", Id: "user-1", Email: "u@example.com", PrimaryRole: role, Roles: []port.UserRole{role}}
}

// Actions whose constant VALUE matches a policy action name. (ActionCreateCart
// is deliberately excluded: its value is "create_cart" but policy.cedar lists
// Action::"CreateCart" — see TestActionNameMismatch_CreateCart.)
var ownerActions = []port.Action{
	port.ActionCreateOrg, port.ActionCreateOrder, port.ActionAddProductToCart,
	port.ActionRemoveItemFromCart, port.ActionProcessWebhook, port.ActionCreateSession,
	port.ActionUpdateSubscription, port.ActionPauseSubscription, port.ActionResumeSubscription,
	port.ActionHealthcheck,
}

var memberActions = []port.Action{
	port.ActionAddProductToCart, port.ActionRemoveItemFromCart, port.ActionCreateSession,
	port.ActionUpdateSubscription, port.ActionPauseSubscription, port.ActionResumeSubscription,
	port.ActionHealthcheck,
}

// ownerOnlyActions are in the owner permit list but NOT the member list.
var ownerOnlyActions = []port.Action{port.ActionCreateOrg, port.ActionCreateOrder, port.ActionProcessWebhook}

// The admin rule is unconditional: admins may do anything.
func TestAdmin_AllowsEverything(t *testing.T) {
	authz := newTestAuthz(t)
	admin := user(port.RoleAdmin)

	for _, action := range ownerActions {
		t.Run(string(action), func(t *testing.T) {
			assert.True(t, authz.Enforce(admin, action, ""), "admin must be permitted %q", action)
		})
	}
	assert.True(t, authz.Enforce(admin, port.Action("CompletelyUnknownAction"), ""), "admin permit is unconditional")
	assert.True(t, authz.Enforce(admin, port.ActionGetDunningCampaign, "campaign-123"), "resource id doesn't change admin verdict")
}

// Owner is permitted every action in its policy list (same-org, which the
// adapter asserts) and denied actions outside it.
func TestOwner_AllowedItsActions(t *testing.T) {
	authz := newTestAuthz(t)
	owner := user(port.RoleOwner)

	for _, action := range ownerActions {
		t.Run("allowed/"+string(action), func(t *testing.T) {
			assert.True(t, authz.Enforce(owner, action, ""), "owner should be permitted %q", action)
		})
	}
	assert.False(t, authz.Enforce(owner, port.Action("NoSuchAction"), ""), "owner denied an action outside the permit list")
}

// Member is scoped to a subset; owner-only actions are denied.
func TestMember_ScopedToSubset(t *testing.T) {
	authz := newTestAuthz(t)
	member := user(port.RoleMember)

	for _, action := range memberActions {
		t.Run("allowed/"+string(action), func(t *testing.T) {
			assert.True(t, authz.Enforce(member, action, ""), "member should be permitted %q", action)
		})
	}
	for _, action := range ownerOnlyActions {
		t.Run("denied/"+string(action), func(t *testing.T) {
			assert.False(t, authz.Enforce(member, action, ""), "member should be denied owner-only %q", action)
		})
	}
}

// Roles referenced by no permit rule are denied (default-deny).
func TestSupportAndUnknownRoles_Denied(t *testing.T) {
	authz := newTestAuthz(t)
	for _, role := range []port.UserRole{port.RoleSupport, port.UserRole("intruder")} {
		for _, action := range []port.Action{port.ActionCreateOrder, port.ActionAddProductToCart, port.ActionHealthcheck} {
			t.Run(string(role)+"/"+string(action), func(t *testing.T) {
				assert.False(t, authz.Enforce(user(role), action, ""), "role %q has no permit rule and must be denied", role)
			})
		}
	}
}

// TestActionNameMismatch_CreateCart documents a real inconsistency: the action
// constant value ("create_cart") does not match the policy action name
// ("CreateCart"), so an owner is denied CreateCart even though the policy lists
// it. Flagged rather than worked around.
func TestActionNameMismatch_CreateCart(t *testing.T) {
	authz := newTestAuthz(t)
	assert.NotEqual(t, "CreateCart", string(port.ActionCreateCart), "constant value diverges from the policy action name")
	assert.False(t, authz.Enforce(user(port.RoleOwner), port.ActionCreateCart, ""),
		"owner is denied CreateCart purely due to the name mismatch (policy says \"CreateCart\", code emits \"create_cart\")")
}

// TestPolicy_IsOrgScoped proves the policy itself enforces
// principal.org_id == resource.org_id when the entities carry distinct orgs.
// The adapter currently asserts both org_ids as the caller's (resources aren't
// passed with an owning org), so cross-org isolation is enforced at the data
// layer; this test pins that the policy is capable of org-scoping should the
// adapter ever supply the resource's real org.
func TestPolicy_IsOrgScoped(t *testing.T) {
	raw, err := os.ReadFile(repoRootPolicyPath(t))
	require.NoError(t, err)
	ps, err := cedar.NewPolicySetFromBytes("policy.cedar", raw)
	require.NoError(t, err)
	assert.Equal(t, 3, len(ps.Map()), "all three permit rules load")

	principal := types.NewEntityUID("Role", "owner")
	resource := types.NewEntityUID("Resource", "")
	build := func(pOrg, rOrg string) cedar.EntityMap {
		return cedar.EntityMap{
			principal: types.Entity{UID: principal, Attributes: types.NewRecord(types.RecordMap{"org_id": types.String(pOrg)})},
			resource:  types.Entity{UID: resource, Attributes: types.NewRecord(types.RecordMap{"org_id": types.String(rOrg)})},
		}
	}
	req := cedar.Request{Principal: principal, Action: types.NewEntityUID("Action", "UpdateSubscription"), Resource: resource, Context: types.NewRecord(types.RecordMap{})}

	allow, _ := ps.IsAuthorized(build("org-1", "org-1"), req)
	assert.True(t, bool(allow), "owner permitted within their own org")
	deny, _ := ps.IsAuthorized(build("org-1", "org-2"), req)
	assert.False(t, bool(deny), "owner denied on another org's resource")
}

// TestPolicy_AdminIsOrgScoped pins the defense-in-depth fix: the admin rule
// now carries `when { principal.org_id == resource.org_id }`. The previous
// version had no constraint, so any subject with the "admin" role had
// unconditional cross-tenant authority. This test reaches into the raw
// policy (bypassing the adapter, which coerces both org_ids to the
// caller's) and proves the rule denies cross-org access at the policy
// level.
func TestPolicy_AdminIsOrgScoped(t *testing.T) {
	raw, err := os.ReadFile(repoRootPolicyPath(t))
	require.NoError(t, err)
	ps, err := cedar.NewPolicySetFromBytes("policy.cedar", raw)
	require.NoError(t, err)

	admin := types.NewEntityUID("Role", "admin")
	resource := types.NewEntityUID("Resource", "")
	build := func(pOrg, rOrg string) cedar.EntityMap {
		return cedar.EntityMap{
			admin:    types.Entity{UID: admin, Attributes: types.NewRecord(types.RecordMap{"org_id": types.String(pOrg)})},
			resource: types.Entity{UID: resource, Attributes: types.NewRecord(types.RecordMap{"org_id": types.String(rOrg)})},
		}
	}
	req := cedar.Request{Principal: admin, Action: types.NewEntityUID("Action", "AnythingAtAll"), Resource: resource, Context: types.NewRecord(types.RecordMap{})}

	allow, _ := ps.IsAuthorized(build("org-1", "org-1"), req)
	assert.True(t, bool(allow), "admin permitted within their own org")
	deny, _ := ps.IsAuthorized(build("org-1", "other-org"), req)
	assert.False(t, bool(deny), "admin denied on another org's resource — the platform-admin escape hatch is closed")
}
