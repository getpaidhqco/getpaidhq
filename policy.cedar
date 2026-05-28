// Permit admin users to perform all actions on all resources
permit(
    principal == Role::"admin",
    action,
    resource
);


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
        Action::"Healthcheck"
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
        Action::"Healthcheck"
    ],
    resource

)when { principal.org_id == resource.org_id };