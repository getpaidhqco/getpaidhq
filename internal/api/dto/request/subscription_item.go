package request

// CreateSubscriptionItemRequest represents a request to create a subscription item
type CreateSubscriptionItemRequest struct {
	OrgId          string            `json:"org_id"`
	SubscriptionId string            `json:"subscription_id" binding:"required"`
	PriceId        string            `json:"price_id" binding:"required"`
	Name           string            `json:"name" binding:"required"`
	Description    string            `json:"description"`
	Quantity       int               `json:"quantity" binding:"min=1"`
	HasUsage       bool              `json:"has_usage"`
	UsageType      string            `json:"usage_type"`
	AggregationType string           `json:"aggregation_type"`
	Metadata       map[string]string `json:"metadata"`
}

// UpdateSubscriptionItemRequest represents a request to update a subscription item
type UpdateSubscriptionItemRequest struct {
	OrgId          string            `json:"org_id"`
	Id             string            `json:"id" binding:"required"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Quantity       int               `json:"quantity" binding:"min=1"`
	HasUsage       bool              `json:"has_usage"`
	UsageType      string            `json:"usage_type"`
	AggregationType string           `json:"aggregation_type"`
	Metadata       map[string]string `json:"metadata"`
}

// PauseSubscriptionItemRequest represents a request to pause a subscription item
type PauseSubscriptionItemRequest struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id" binding:"required"`
}

// ResumeSubscriptionItemRequest represents a request to resume a subscription item
type ResumeSubscriptionItemRequest struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id" binding:"required"`
}

// CancelSubscriptionItemRequest represents a request to cancel a subscription item
type CancelSubscriptionItemRequest struct {
	OrgId  string `json:"org_id"`
	Id     string `json:"id" binding:"required"`
	Reason string `json:"reason"`
}