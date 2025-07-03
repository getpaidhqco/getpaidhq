package entities

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/domain/errors"
	"payloop/internal/lib"
	"time"
)

type Price struct {
	OrgId              string                 `json:"org_id"`
	Id                 string                 `json:"id"`
	VariantId          string                 `json:"variant_id"`
	Label              string                 `json:"label"`
	Category           prices.PriceCategory   `json:"category"`
	Scheme             prices.PriceScheme     `json:"scheme"`
	Cycles             int                    `json:"cycles"`
	Currency           common.Currency        `json:"currency"`
	UnitPrice          int64                  `json:"unit_price"`
	MinPrice           int64                  `json:"min_price"`
	SuggestedPrice     int64                  `json:"suggested_price"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	TrialInterval      prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty"`
	TaxCode            string                 `json:"tax_code"`

	// Usage-based billing fields
	HasUsage           bool                    `json:"has_usage"`
	UsageType          prices.UsageType        `json:"usage_type,omitempty"`
	UnitType           prices.UnitType         `json:"unit_type,omitempty"`
	AggregationType    prices.AggregationType  `json:"aggregation_type,omitempty"`
	PercentageRate     float64                 `json:"percentage_rate,omitempty"`
	FixedFee           int64                   `json:"fixed_fee,omitempty"`
	OverageUnitPrice   int64                   `json:"overage_unit_price,omitempty"`
	IncludedUsage      int64                   `json:"included_usage,omitempty"`
	UsageLimit         int64                   `json:"usage_limit,omitempty"`

	// Tier configuration
	Tiers              []PriceTier             `json:"tiers,omitempty"`

	Metadata           map[string]string       `json:"metadata"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

// Factory function to create a Price with default values and validation
func NewPrice(orgId, variantId string, input CreatePriceInput) (Price, error) {
	// Validate required fields
	if orgId == "" {
		return Price{}, errors.ErrMissingOrgId
	}
	if variantId == "" {
		return Price{}, errors.ErrMissingVariantId
	}
	if input.Currency == "" {
		return Price{}, errors.ErrMissingCurrency
	}

	// Set defaults for billing intervals
	if input.BillingInterval == "" {
		input.BillingInterval = prices.BillingIntervalNone
	}
	if input.TrialInterval == "" {
		input.TrialInterval = prices.BillingIntervalNone
	}

	// Generate a price ID
	priceId := lib.GenerateId("price")

	// Create the price entity
	price := Price{
		OrgId:              orgId,
		Id:                 priceId,
		Label:              input.Label,
		VariantId:          variantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           common.Currency(input.Currency),
		UnitPrice:          input.UnitPrice,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		Metadata:           input.Metadata,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	// Configure usage fields based on HasUsage flag
	if input.HasUsage {
		if err := price.configureUsageBilling(input); err != nil {
			return Price{}, err
		}
	} else {
		price.clearUsageFields()
	}

	// Convert CreatePriceTierInput to PriceTier
	var tiers []PriceTier
	for _, tierInput := range input.Tiers {
		tiers = append(tiers, NewPriceTier(CreatePriceTierInput{
			OrgId:       orgId,
			PriceId:     priceId,
			Tier:        tierInput.Tier,
			FromQty:     tierInput.FromQty,
			ToQty:       tierInput.ToQty,
			UnitPrice:   tierInput.UnitPrice,
			Description: tierInput.Description,
		}))
	}
	price.Tiers = tiers

	// Validate the price entity
	if err := price.Validate(); err != nil {
		return Price{}, err
	}

	return price, nil
}

// Validate performs validation on the Price entity based on its category
func (p Price) Validate() error {
	// Validate percentage-based pricing
	if p.PercentageRate > 0 && p.UnitType != prices.UnitTypeTransactions && p.UnitType != "cents" && p.UnitType != "dollars" {
		return errors.NewValidationError("percentageRate", "percentage rate only valid for transaction-based units")
	}

	// Validate price category
	switch p.Category {
	case prices.PriceCategorySubscription:
		return p.validateSubscriptionPricing()
	case prices.PriceCategoryUsage:
		return p.validateUsagePricing()
	case prices.PriceCategoryHybrid:
		return p.validateHybridPricing()
	case prices.Free:
		return p.validateFreePricing()
	case prices.OneTime, prices.Variable:
		// These categories might have their own validation in the future
		return nil
	default:
		return errors.ErrInvalidPriceCategory
	}
}

// validateSubscriptionPricing validates traditional subscription pricing
func (p Price) validateSubscriptionPricing() error {
	// Ensure HasUsage is false
	if p.HasUsage {
		return errors.NewValidationError("hasUsage", "subscription pricing cannot have usage-based billing")
	}

	// Validate UnitPrice is not negative
	if p.UnitPrice < 0 {
		return errors.NewValidationError("unitPrice", "unit price cannot be negative")
	}

	// Require BillingInterval if UnitPrice > 0
	if p.UnitPrice > 0 && p.BillingInterval == prices.BillingIntervalNone {
		return errors.NewValidationError("billingInterval", "billing interval is required for paid subscriptions")
	}

	return nil
}

// validateUsagePricing validates pure usage-based pricing
func (p Price) validateUsagePricing() error {
	// Ensure HasUsage is true
	if !p.HasUsage {
		return errors.NewValidationError("hasUsage", "usage pricing requires hasUsage to be true")
	}

	// Require UsageType, UnitType, and AggregationType
	if p.UsageType == "" {
		return errors.NewValidationError("usageType", "usage type is required for usage-based pricing")
	}
	if p.UnitType == "" {
		return errors.NewValidationError("unitType", "unit type is required for usage-based pricing")
	}
	if p.AggregationType == "" {
		return errors.NewValidationError("aggregationType", "aggregation type is required for usage-based pricing")
	}

	// For transactions (UnitTypeTransactions): require either PercentageRate or UnitPrice
	if p.UnitType == prices.UnitTypeTransactions && p.PercentageRate == 0 && p.UnitPrice == 0 {
		return errors.NewValidationError("pricing", "transaction pricing requires either percentage rate or unit price")
	}

	// For other types: require UnitPrice or PercentageRate
	if p.UnitType != prices.UnitTypeTransactions && p.UnitPrice == 0 && p.PercentageRate == 0 {
		return errors.NewValidationError("pricing", "usage pricing requires either unit price or percentage rate")
	}

	// Validate tiers if scheme is tiered/volume
	if p.Scheme == prices.Tiered || p.Scheme == prices.Volume {
		return p.validateTiers()
	}

	return nil
}

// validateHybridPricing validates hybrid pricing (base + usage)
func (p Price) validateHybridPricing() error {
	// Ensure HasUsage is true
	if !p.HasUsage {
		return errors.NewValidationError("hasUsage", "hybrid pricing requires hasUsage to be true")
	}

	// Require positive UnitPrice (base amount)
	if p.UnitPrice <= 0 {
		return errors.NewValidationError("unitPrice", "hybrid pricing requires a positive base price")
	}

	// Require OverageUnitPrice or PercentageRate for overages
	if p.OverageUnitPrice == 0 && p.PercentageRate == 0 {
		return errors.NewValidationError("pricing", "hybrid pricing requires either overage unit price or percentage rate")
	}

	// Require usage configuration fields
	if p.UsageType == "" {
		return errors.NewValidationError("usageType", "usage type is required for hybrid pricing")
	}
	if p.UnitType == "" {
		return errors.NewValidationError("unitType", "unit type is required for hybrid pricing")
	}
	if p.AggregationType == "" {
		return errors.NewValidationError("aggregationType", "aggregation type is required for hybrid pricing")
	}

	// Validate tiers if scheme is tiered/volume
	if p.Scheme == prices.Tiered || p.Scheme == prices.Volume {
		return p.validateTiers()
	}

	return nil
}

// validateFreePricing validates free tier pricing
func (p Price) validateFreePricing() error {
	// Ensure UnitPrice is 0
	if p.UnitPrice != 0 {
		return errors.NewValidationError("unitPrice", "free tier must have zero unit price")
	}

	// Ensure OverageUnitPrice is 0
	if p.OverageUnitPrice != 0 {
		return errors.NewValidationError("overageUnitPrice", "free tier cannot have overage charges")
	}

	// No charges allowed for free tier
	if p.PercentageRate != 0 || p.FixedFee != 0 {
		return errors.NewValidationError("pricing", "free tier cannot have any charges")
	}

	return nil
}

// validateTiers validates tier configuration
func (p Price) validateTiers() error {
	// Only allow tiers for tiered/volume schemes
	if p.Scheme != prices.Tiered && p.Scheme != prices.Volume && len(p.Tiers) > 0 {
		return errors.NewValidationError("tiers", "tiers are only allowed for tiered or volume pricing schemes")
	}

	// Require at least one tier for tiered/volume schemes
	if (p.Scheme == prices.Tiered || p.Scheme == prices.Volume) && len(p.Tiers) == 0 {
		return errors.NewValidationError("tiers", "at least one tier is required for tiered or volume pricing")
	}

	// No validation needed if no tiers
	if len(p.Tiers) == 0 {
		return nil
	}

	// First tier must start from quantity 1
	if p.Tiers[0].FromQty != 1 {
		return errors.NewValidationError("tiers", "first tier must start from quantity 1")
	}

	// Validate tier continuity (no gaps in quantity ranges)
	for i := 0; i < len(p.Tiers)-1; i++ {
		currentTier := p.Tiers[i]
		nextTier := p.Tiers[i+1]

		// Check for negative unit prices
		if currentTier.UnitPrice < 0 {
			return errors.NewValidationError("tiers", "tier unit price cannot be negative")
		}

		// If current tier has a ToQty, it should connect with the next tier's FromQty
		if currentTier.ToQty != nil {
			if *currentTier.ToQty+1 != nextTier.FromQty {
				return errors.NewValidationError("tiers", "tiers must be continuous with no gaps")
			}
		} else {
			// If ToQty is nil (unlimited), this should be the last tier
			if i < len(p.Tiers)-1 {
				return errors.NewValidationError("tiers", "only the last tier can have unlimited quantity")
			}
		}
	}

	// Check the last tier's unit price
	if p.Tiers[len(p.Tiers)-1].UnitPrice < 0 {
		return errors.NewValidationError("tiers", "tier unit price cannot be negative")
	}

	return nil
}

// configureUsageBilling configures usage fields based on input
func (p *Price) configureUsageBilling(input CreatePriceInput) error {
	p.HasUsage = true
	p.UsageType = prices.UsageType(input.UsageType)
	p.UnitType = prices.UnitType(input.UnitType)
	p.AggregationType = prices.AggregationType(input.AggregationType)
	p.PercentageRate = input.PercentageRate
	p.FixedFee = input.FixedFee
	p.OverageUnitPrice = input.OverageUnitPrice
	p.IncludedUsage = input.IncludedUsage
	p.UsageLimit = input.UsageLimit

	// Validate usage configuration
	if p.UsageType == "" || p.UnitType == "" || p.AggregationType == "" {
		return errors.ErrInvalidUsageConfiguration
	}

	return nil
}

// clearUsageFields clears all usage-related fields when HasUsage is false
func (p *Price) clearUsageFields() {
	p.HasUsage = false
	p.UsageType = ""
	p.UnitType = ""
	p.AggregationType = ""
	p.PercentageRate = 0
	p.FixedFee = 0
	p.OverageUnitPrice = 0
	p.IncludedUsage = 0
	p.UsageLimit = 0
}

type CreatePriceInput struct {
	OrgId              string                 `json:"org_id"`
	Label              string                 `json:"label"`
	VariantId          string                 `json:"variant_id"`
	Category           prices.PriceCategory   `json:"category"`
	Scheme             prices.PriceScheme     `json:"scheme"`
	Cycles             int                    `json:"cycles"`
	Currency           string                 `json:"currency"`
	UnitPrice          int64                  `json:"unit_price"`
	MinPrice           int64                  `json:"min_price"`
	SuggestedPrice     int64                  `json:"suggested_price"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	TrialInterval      prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty"`
	TaxCode            string                 `json:"tax_code"`

	// Usage-based billing fields
	HasUsage           bool                   `json:"has_usage"`
	UsageType          string                 `json:"usage_type,omitempty"`
	UnitType           string                 `json:"unit_type,omitempty"`
	AggregationType    string                 `json:"aggregation_type,omitempty"`
	PercentageRate     float64                `json:"percentage_rate,omitempty"`
	FixedFee           int64                  `json:"fixed_fee,omitempty"`
	OverageUnitPrice   int64                  `json:"overage_unit_price,omitempty"`
	IncludedUsage      int64                  `json:"included_usage,omitempty"`
	UsageLimit         int64                  `json:"usage_limit,omitempty"`

	// Tier configuration
	Tiers              []CreatePriceTierInput `json:"tiers,omitempty"`

	Metadata           map[string]string      `json:"metadata"`
}
