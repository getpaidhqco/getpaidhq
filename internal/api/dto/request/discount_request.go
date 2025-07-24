package request

// CreateDiscountRequest represents the request for creating a discount
type CreateDiscountRequest struct {
	Name           string                 `json:"name" binding:"required"`
	Type           string                 `json:"type" binding:"required,oneof=fixed percentage"`
	Value          int                    `json:"value" binding:"required"`
	Code           string                 `json:"code,omitempty"`
	StartsAt       string                 `json:"starts_at,omitempty" binding:"omitempty,rfc3339"`
	EndsAt         string                 `json:"ends_at,omitempty" binding:"omitempty,rfc3339"`
	MaxRedemptions int                    `json:"max_redemptions,omitempty"`
	Recurring      string                 `json:"recurring" binding:"required,oneof=once forever cycles"`
	Cycles         int                    `json:"cycles,omitempty" binding:"omitempty,gt=0"`
	Currency       string                 `json:"currency,omitempty"`
	Active         bool                   `json:"active"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateDiscountRequest represents the request for updating a discount
type UpdateDiscountRequest struct {
	Name           string                 `json:"name,omitempty"`
	Type           string                 `json:"type,omitempty" binding:"omitempty,oneof=fixed percentage"`
	Value          *int                   `json:"value,omitempty"`
	Code           string                 `json:"code,omitempty"`
	StartsAt       string                 `json:"starts_at,omitempty" binding:"omitempty,rfc3339"`
	EndsAt         string                 `json:"ends_at,omitempty" binding:"omitempty,rfc3339"`
	MaxRedemptions *int                   `json:"max_redemptions,omitempty"`
	Recurring      string                 `json:"recurring,omitempty" binding:"omitempty,oneof=once forever cycles"`
	Cycles         *int                   `json:"cycles,omitempty" binding:"omitempty,gt=0"`
	Currency       string                 `json:"currency,omitempty"`
	Active         *bool                  `json:"active,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ValidateDiscountCodeRequest represents the request for validating a discount code
type ValidateDiscountCodeRequest struct {
	Code       string `json:"code" binding:"required"`
	CustomerId string `json:"customer_id,omitempty"`
	Amount     int    `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
}

// ApplyDiscountRequest represents the request for applying a discount
type ApplyDiscountRequest struct {
	DiscountId   string                 `json:"discount_id" binding:"required"`
	CustomerId   string                 `json:"customer_id" binding:"required"`
	ResourceType string                 `json:"resource_type" binding:"required,oneof=subscription invoice payment checkout_session"`
	ResourceId   string                 `json:"resource_id" binding:"required"`
	Amount       int                    `json:"amount" binding:"required,gt=0"`
	Currency     string                 `json:"currency" binding:"required"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
