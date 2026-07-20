// Admins may perform any action — but only within their own org. The
// previous version had no `when { ... }` clause, which made `admin` a
// platform-wide super-role; one mis-issued token would have been a full
// cross-tenant breach.
permit(
    principal == Role::"admin",
    action,
    resource
)
when {
    principal.org_id == resource.org_id
};


permit(
    principal,
    action in [
        Action::"CreateOrg",
        Action::"CreateCart",
        Action::"CreateOrder",
        Action::"AddProductToCart",
        Action::"RemoveItemFromCart",
        Action::"ProcessWebhook",
        Action::"CreateSession",
        Action::"UpdateSubscription",
        Action::"PauseSubscription",
        Action::"ResumeSubscription",
        Action::"CancelSubscription",
        Action::"UpdateBillingAnchor",
        Action::"Healthcheck",
        Action::"CreateApiKey",
        Action::"ListApiKeys",
        Action::"DeleteApiKey"
    ],
    resource
)
when {
    principal == Role::"owner" &&
    principal.org_id == resource.org_id
};

permit(
    principal == Role::"member",
    action in [
        Action::"AddProductToCart",
        Action::"RemoveItemFromCart",
        Action::"CreateSession",
        Action::"UpdateSubscription",
        Action::"PauseSubscription",
        Action::"ResumeSubscription",
        Action::"CancelSubscription",
        Action::"UpdateBillingAnchor",
        Action::"ReadUsage",
        Action::"Healthcheck"
    ],
    resource

)when { principal.org_id == resource.org_id };

// ---------------------------------------------------------------------------
// Limen organization plugin. The plugin ships zero policy (fail-closed); these
// rules ARE the authorization for org management. Roles are derived from
// membership facts by internal/adapter/limenauth: the org creator acts as
// "owner", other members as "member", non-members as "none" (no rule → deny).
// Action names are the plugin's stable Action constants, used verbatim.
// ---------------------------------------------------------------------------

permit(
    principal == Role::"owner",
    action in [
        Action::"organization.update",
        Action::"organization.delete",
        Action::"member.remove",
        Action::"invitation.create",
        Action::"invitation.revoke"
    ],
    resource
)
when { principal.org_id == resource.org_id };

permit(
    principal == Role::"member",
    action in [
        Action::"invitation.create"
    ],
    resource
)
when { principal.org_id == resource.org_id };
