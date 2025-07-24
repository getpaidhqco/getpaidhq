package dto

// CreateDiscountInput represents the input for creating a discount
type CreateDiscountInput struct {
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`           // "fixed" or "percentage"
	Value          int                    `json:"value"`          // For fixed: amount in smallest currency unit. For percentage: 0-100 (e.g., 20 = 20%)
	Code           string                 `json:"code,omitempty"` // Optional discount code
	StartsAt       string                 `json:"starts_at,omitempty"`
	EndsAt         string                 `json:"ends_at,omitempty"`
	MaxRedemptions int                    `json:"max_redemptions,omitempty"` // Maximum number of times this discount can be redeemed
	Recurring      string                 `json:"recurring"`                 // "once", "forever", "cycles"
	Cycles         int                    `json:"cycles,omitempty"`          // Number of billing cycles when recurring is "cycles"
	Currency       string                 `json:"currency,omitempty"`        // Required for fixed discounts
	Active         bool                   `json:"active"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateDiscountInput represents the input for updating a discount
type UpdateDiscountInput struct {
	Name           string                 `json:"name,omitempty"`
	Type           string                 `json:"type,omitempty"`           // "fixed" or "percentage"
	Value          *int                   `json:"value,omitempty"`          // For fixed: amount in smallest currency unit. For percentage: 0-100 (e.g., 20 = 20%)
	Code           string                 `json:"code,omitempty"`           // Optional discount code
	StartsAt       string                 `json:"starts_at,omitempty"`
	EndsAt         string                 `json:"ends_at,omitempty"`
	MaxRedemptions *int                   `json:"max_redemptions,omitempty"` // Maximum number of times this discount can be redeemed
	Recurring      string                 `json:"recurring,omitempty"`       // "once", "forever", "cycles"
	Cycles         *int                   `json:"cycles,omitempty"`          // Number of billing cycles when recurring is "cycles"
	Currency       string                 `json:"currency,omitempty"`        // Required for fixed discounts
	Active         *bool                  `json:"active,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ValidateDiscountCodeInput represents the input for validating a discount code
type ValidateDiscountCodeInput struct {
	Code       string `json:"code"`
	CustomerId string `json:"customer_id,omitempty"`
	Amount     int    `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
}

// DiscountValidationResult represents the result of validating a discount code
type DiscountValidationResult struct {
	Valid          bool   `json:"valid"`
	DiscountId     string `json:"discount_id,omitempty"`
	DiscountAmount int    `json:"discount_amount,omitempty"`
	Message        string `json:"message,omitempty"`
}

// ApplyDiscountInput represents the input for applying a discount
type ApplyDiscountInput struct {
	DiscountId   string                 `json:"discount_id"`
	CustomerId   string                 `json:"customer_id"`
	ResourceType string                 `json:"resource_type"` // "subscription", "invoice", "payment", "checkout_session"
	ResourceId   string                 `json:"resource_id"`
	Amount       int                    `json:"amount"`       // Original amount before discount
	Currency     string                 `json:"currency"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// DiscountRedemptionResult represents the result of applying a discount
type DiscountRedemptionResult struct {
	RedemptionId    string `json:"redemption_id"`
	DiscountId      string `json:"discount_id"`
	DiscountAmount  int    `json:"discount_amount"`
	OriginalAmount  int    `json:"original_amount"`
	DiscountedAmount int   `json:"discounted_amount"`
	Currency        string `json:"currency"`
}
