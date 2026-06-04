package service

import "getpaidhq/internal/core/domain"

// SubscriptionDetails is the composed read model for "show me a subscription"
// queries. The HTTP layer renders SubscriptionResponse from this; do not put
// adapter types in here. Cross-aggregate composition lives in the service.
type SubscriptionDetails struct {
	Subscription domain.Subscription
	Customer     domain.Customer
}
