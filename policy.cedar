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
