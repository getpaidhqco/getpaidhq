package response

import (
	"time"
)

// DiscountResponse represents a discount in API responses
type DiscountResponse struct {
	Id             string                 `json:"id"`
	OrgId          string                 `json:"org_id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Value          int                    `json:"value"`
	Code           string                 `json:"code,omitempty"`
	StartsAt       *time.Time             `json:"starts_at,omitempty"`
	EndsAt         *time.Time             `json:"ends_at,omitempty"`
	MaxRedemptions int                    `json:"max_redemptions,omitempty"`
	Recurring      string                 `json:"recurring"`
	Cycles         int                    `json:"cycles,omitempty"`
	Currency       string                 `json:"currency,omitempty"`
	Active         bool                   `json:"active"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// DiscountListResponse represents a paginated list of discounts
type DiscountListResponse struct {
	Items      []DiscountResponse `json:"items"`
	TotalCount int                `json:"total_count"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	HasMore    bool               `json:"has_more"`
}

// DiscountRedemptionResponse represents a discount redemption in API responses
type DiscountRedemptionResponse struct {
	Id             string                 `json:"id"`
	OrgId          string                 `json:"org_id"`
	DiscountId     string                 `json:"discount_id"`
	CustomerId     string                 `json:"customer_id"`
	ResourceType   string                 `json:"resource_type"`
	ResourceId     string                 `json:"resource_id"`
	DiscountAmount int                    `json:"discount_amount"`
	Currency       string                 `json:"currency"`
	CreatedAt      time.Time              `json:"created_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// DiscountRedemptionListResponse represents a paginated list of discount redemptions
type DiscountRedemptionListResponse struct {
	Items      []DiscountRedemptionResponse `json:"items"`
	TotalCount int                          `json:"total_count"`
	Page       int                          `json:"page"`
	PageSize   int                          `json:"page_size"`
	HasMore    bool                         `json:"has_more"`
}

// DiscountValidationResponse represents the result of validating a discount code
type DiscountValidationResponse struct {
	Valid          bool   `json:"valid"`
	DiscountId     string `json:"discount_id,omitempty"`
	DiscountAmount int    `json:"discount_amount,omitempty"`
	Message        string `json:"message,omitempty"`
}

// DiscountRedemptionResultResponse represents the result of applying a discount
type DiscountRedemptionResultResponse struct {
	RedemptionId     string `json:"redemption_id"`
	DiscountId       string `json:"discount_id"`
	DiscountAmount   int    `json:"discount_amount"`
	OriginalAmount   int    `json:"original_amount"`
	DiscountedAmount int    `json:"discounted_amount"`
	Currency         string `json:"currency"`
}