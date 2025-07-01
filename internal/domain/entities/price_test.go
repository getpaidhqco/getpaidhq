package entities

import (
	"payloop/internal/domain/entities/prices"
	"payloop/internal/domain/errors"
	"strings"
	"testing"
)

func TestNewPrice_RequiredFields(t *testing.T) {
	tests := []struct {
		name      string
		orgId     string
		variantId string
		input     CreatePriceInput
		wantErr   bool
		errType   error
	}{
		{
			name:      "Valid input with all required fields",
			orgId:     "org_123",
			variantId: "var_123",
			input: CreatePriceInput{
				Currency: "USD",
				Category: prices.PriceCategorySubscription,
			},
			wantErr: false,
		},
		{
			name:      "Missing orgId",
			orgId:     "",
			variantId: "var_123",
			input: CreatePriceInput{
				Currency: "USD",
			},
			wantErr: true,
			errType: errors.ErrMissingOrgId,
		},
		{
			name:      "Missing variantId",
			orgId:     "org_123",
			variantId: "",
			input: CreatePriceInput{
				Currency: "USD",
			},
			wantErr: true,
			errType: errors.ErrMissingVariantId,
		},
		{
			name:      "Missing currency",
			orgId:     "org_123",
			variantId: "var_123",
			input: CreatePriceInput{
				Category: prices.PriceCategorySubscription,
			},
			wantErr: true,
			errType: errors.ErrMissingCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPrice(tt.orgId, tt.variantId, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPrice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("NewPrice() error = %v, want error type %v", err, tt.errType)
			}
		})
	}
}

func TestPrice_ValidateSubscriptionPricing(t *testing.T) {
	tests := []struct {
		name    string
		price   Price
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid subscription pricing",
			price: Price{
				Category:        prices.PriceCategorySubscription,
				HasUsage:        false,
				UnitPrice:       1000,
				BillingInterval: prices.BillingIntervalMonth,
			},
			wantErr: false,
		},
		{
			name: "Subscription with HasUsage=true",
			price: Price{
				Category: prices.PriceCategorySubscription,
				HasUsage: true,
			},
			wantErr: true,
			errMsg:  "subscription pricing cannot have usage-based billing",
		},
		{
			name: "Subscription with negative UnitPrice",
			price: Price{
				Category:  prices.PriceCategorySubscription,
				HasUsage:  false,
				UnitPrice: -100,
			},
			wantErr: true,
			errMsg:  "unit price cannot be negative",
		},
		{
			name: "Paid subscription without billing interval",
			price: Price{
				Category:        prices.PriceCategorySubscription,
				HasUsage:        false,
				UnitPrice:       1000,
				BillingInterval: prices.BillingIntervalNone,
			},
			wantErr: true,
			errMsg:  "billing interval is required for paid subscriptions",
		},
		{
			name: "Free subscription without billing interval",
			price: Price{
				Category:        prices.PriceCategorySubscription,
				HasUsage:        false,
				UnitPrice:       0,
				BillingInterval: prices.BillingIntervalNone,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.price.validateSubscriptionPricing()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSubscriptionPricing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateSubscriptionPricing() error = %v, want error message containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestPrice_ValidateUsagePricing(t *testing.T) {
	tests := []struct {
		name    string
		price   Price
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid usage pricing with unit price",
			price: Price{
				Category:         prices.PriceCategoryUsage,
				HasUsage:         true,
				UsageType:        prices.UsageTypeMetered,
				UnitType:         prices.UnitTypeApiCalls,
				AggregationType:  prices.AggregationTypeSum,
				OverageUnitPrice: 10,
			},
			wantErr: false,
		},
		{
			name: "Valid usage pricing with percentage rate",
			price: Price{
				Category:        prices.PriceCategoryUsage,
				HasUsage:        true,
				UsageType:       prices.UsageTypeMetered,
				UnitType:        prices.UnitTypeTransactions,
				AggregationType: prices.AggregationTypeSum,
				PercentageRate:  2.5,
			},
			wantErr: false,
		},
		{
			name: "Usage pricing with HasUsage=false",
			price: Price{
				Category: prices.PriceCategoryUsage,
				HasUsage: false,
			},
			wantErr: true,
			errMsg:  "usage pricing requires hasUsage to be true",
		},
		{
			name: "Usage pricing without UsageType",
			price: Price{
				Category:         prices.PriceCategoryUsage,
				HasUsage:         true,
				UnitType:         prices.UnitTypeApiCalls,
				AggregationType:  prices.AggregationTypeSum,
				OverageUnitPrice: 10,
			},
			wantErr: true,
			errMsg:  "usage type is required for usage-based pricing",
		},
		{
			name: "Usage pricing without UnitType",
			price: Price{
				Category:         prices.PriceCategoryUsage,
				HasUsage:         true,
				UsageType:        prices.UsageTypeMetered,
				AggregationType:  prices.AggregationTypeSum,
				OverageUnitPrice: 10,
			},
			wantErr: true,
			errMsg:  "unit type is required for usage-based pricing",
		},
		{
			name: "Usage pricing without AggregationType",
			price: Price{
				Category:         prices.PriceCategoryUsage,
				HasUsage:         true,
				UsageType:        prices.UsageTypeMetered,
				UnitType:         prices.UnitTypeApiCalls,
				OverageUnitPrice: 10,
			},
			wantErr: true,
			errMsg:  "aggregation type is required for usage-based pricing",
		},
		{
			name: "Transaction pricing without rate or price",
			price: Price{
				Category:         prices.PriceCategoryUsage,
				HasUsage:         true,
				UsageType:        prices.UsageTypeMetered,
				UnitType:         prices.UnitTypeTransactions,
				AggregationType:  prices.AggregationTypeSum,
				PercentageRate:   0,
				OverageUnitPrice: 0,
			},
			wantErr: true,
			errMsg:  "transaction pricing requires either percentage rate or unit price",
		},
		{
			name: "Non-transaction usage pricing without rate or price",
			price: Price{
				Category:         prices.PriceCategoryUsage,
				HasUsage:         true,
				UsageType:        prices.UsageTypeMetered,
				UnitType:         prices.UnitTypeApiCalls,
				AggregationType:  prices.AggregationTypeSum,
				PercentageRate:   0,
				OverageUnitPrice: 0,
			},
			wantErr: true,
			errMsg:  "usage pricing requires either unit price or percentage rate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.price.validateUsagePricing()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUsagePricing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateUsagePricing() error = %v, want error message containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestPrice_ValidateHybridPricing(t *testing.T) {
	tests := []struct {
		name    string
		price   Price
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid hybrid pricing",
			price: Price{
				Category:         prices.PriceCategoryHybrid,
				HasUsage:         true,
				UnitPrice:        1000,
				UsageType:        prices.UsageTypeMetered,
				UnitType:         prices.UnitTypeApiCalls,
				AggregationType:  prices.AggregationTypeSum,
				OverageUnitPrice: 10,
				BillingInterval:  prices.BillingIntervalMonth,
			},
			wantErr: false,
		},
		{
			name: "Hybrid pricing with HasUsage=false",
			price: Price{
				Category:  prices.PriceCategoryHybrid,
				HasUsage:  false,
				UnitPrice: 1000,
			},
			wantErr: true,
			errMsg:  "hybrid pricing requires hasUsage to be true",
		},
		{
			name: "Hybrid pricing with zero UnitPrice",
			price: Price{
				Category:         prices.PriceCategoryHybrid,
				HasUsage:         true,
				UnitPrice:        0,
				UsageType:        prices.UsageTypeMetered,
				UnitType:         prices.UnitTypeApiCalls,
				AggregationType:  prices.AggregationTypeSum,
				OverageUnitPrice: 10,
			},
			wantErr: true,
			errMsg:  "hybrid pricing requires a positive base price",
		},
		{
			name: "Hybrid pricing without overage price or percentage",
			price: Price{
				Category:        prices.PriceCategoryHybrid,
				HasUsage:        true,
				UnitPrice:       1000,
				UsageType:       prices.UsageTypeMetered,
				UnitType:        prices.UnitTypeApiCalls,
				AggregationType: prices.AggregationTypeSum,
			},
			wantErr: true,
			errMsg:  "hybrid pricing requires either overage unit price or percentage rate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.price.validateHybridPricing()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHybridPricing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateHybridPricing() error = %v, want error message containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestPrice_ValidateFreePricing(t *testing.T) {
	tests := []struct {
		name    string
		price   Price
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid free pricing",
			price: Price{
				Category:         prices.Free,
				UnitPrice:        0,
				OverageUnitPrice: 0,
				PercentageRate:   0,
				FixedFee:         0,
			},
			wantErr: false,
		},
		{
			name: "Free pricing with non-zero UnitPrice",
			price: Price{
				Category:  prices.Free,
				UnitPrice: 100,
			},
			wantErr: true,
			errMsg:  "free tier must have zero unit price",
		},
		{
			name: "Free pricing with non-zero OverageUnitPrice",
			price: Price{
				Category:         prices.Free,
				UnitPrice:        0,
				OverageUnitPrice: 10,
			},
			wantErr: true,
			errMsg:  "free tier cannot have overage charges",
		},
		{
			name: "Free pricing with non-zero PercentageRate",
			price: Price{
				Category:       prices.Free,
				UnitPrice:      0,
				PercentageRate: 1.5,
			},
			wantErr: true,
			errMsg:  "free tier cannot have any charges",
		},
		{
			name: "Free pricing with non-zero FixedFee",
			price: Price{
				Category:  prices.Free,
				UnitPrice: 0,
				FixedFee:  100,
			},
			wantErr: true,
			errMsg:  "free tier cannot have any charges",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.price.validateFreePricing()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFreePricing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateFreePricing() error = %v, want error message containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestPrice_ValidateTiers(t *testing.T) {
	// Helper function to create a pointer to an int
	intPtr := func(i int) *int {
		return &i
	}

	tests := []struct {
		name    string
		price   Price
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid tiered pricing with continuous tiers",
			price: Price{
				Scheme: prices.Tiered,
				Tiers: []PriceTier{
					{Tier: 1, FromQty: 1, ToQty: intPtr(10), UnitPrice: 10},
					{Tier: 2, FromQty: 11, ToQty: intPtr(100), UnitPrice: 8},
					{Tier: 3, FromQty: 101, ToQty: nil, UnitPrice: 5},
				},
			},
			wantErr: false,
		},
		{
			name: "Tiers for non-tiered/volume scheme",
			price: Price{
				Scheme: prices.Fixed,
				Tiers: []PriceTier{
					{Tier: 1, FromQty: 1, ToQty: intPtr(10), UnitPrice: 10},
				},
			},
			wantErr: true,
			errMsg:  "tiers are only allowed for tiered or volume pricing schemes",
		},
		{
			name: "Tiered pricing without tiers",
			price: Price{
				Scheme: prices.Tiered,
				Tiers:  []PriceTier{},
			},
			wantErr: true,
			errMsg:  "at least one tier is required for tiered or volume pricing",
		},
		{
			name: "First tier doesn't start from 1",
			price: Price{
				Scheme: prices.Tiered,
				Tiers: []PriceTier{
					{Tier: 1, FromQty: 5, ToQty: intPtr(10), UnitPrice: 10},
				},
			},
			wantErr: true,
			errMsg:  "first tier must start from quantity 1",
		},
		{
			name: "Tiers with gap",
			price: Price{
				Scheme: prices.Tiered,
				Tiers: []PriceTier{
					{Tier: 1, FromQty: 1, ToQty: intPtr(10), UnitPrice: 10},
					{Tier: 2, FromQty: 15, ToQty: intPtr(100), UnitPrice: 8}, // Gap between 11-14
				},
			},
			wantErr: true,
			errMsg:  "tiers must be continuous with no gaps",
		},
		{
			name: "Unlimited tier in the middle",
			price: Price{
				Scheme: prices.Tiered,
				Tiers: []PriceTier{
					{Tier: 1, FromQty: 1, ToQty: intPtr(10), UnitPrice: 10},
					{Tier: 2, FromQty: 11, ToQty: nil, UnitPrice: 8},  // Unlimited tier
					{Tier: 3, FromQty: 101, ToQty: nil, UnitPrice: 5}, // Another tier after unlimited
				},
			},
			wantErr: true,
			errMsg:  "only the last tier can have unlimited quantity",
		},
		{
			name: "Tier with negative unit price",
			price: Price{
				Scheme: prices.Tiered,
				Tiers: []PriceTier{
					{Tier: 1, FromQty: 1, ToQty: intPtr(10), UnitPrice: -10}, // Negative price
				},
			},
			wantErr: true,
			errMsg:  "tier unit price cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.price.validateTiers()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTiers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateTiers() error = %v, want error message containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestPrice_ClearUsageFields(t *testing.T) {
	price := Price{
		HasUsage:         true,
		UsageType:        prices.UsageTypeMetered,
		UnitType:         prices.UnitTypeApiCalls,
		AggregationType:  prices.AggregationTypeSum,
		PercentageRate:   2.5,
		FixedFee:         100,
		OverageUnitPrice: 10,
		IncludedUsage:    1000,
		UsageLimit:       5000,
	}

	price.clearUsageFields()

	if price.HasUsage {
		t.Errorf("clearUsageFields() did not clear HasUsage, got %v", price.HasUsage)
	}
	if price.UsageType != "" {
		t.Errorf("clearUsageFields() did not clear UsageType, got %v", price.UsageType)
	}
	if price.UnitType != "" {
		t.Errorf("clearUsageFields() did not clear UnitType, got %v", price.UnitType)
	}
	if price.AggregationType != "" {
		t.Errorf("clearUsageFields() did not clear AggregationType, got %v", price.AggregationType)
	}
	if price.PercentageRate != 0 {
		t.Errorf("clearUsageFields() did not clear PercentageRate, got %v", price.PercentageRate)
	}
	if price.FixedFee != 0 {
		t.Errorf("clearUsageFields() did not clear FixedFee, got %v", price.FixedFee)
	}
	if price.OverageUnitPrice != 0 {
		t.Errorf("clearUsageFields() did not clear OverageUnitPrice, got %v", price.OverageUnitPrice)
	}
	if price.IncludedUsage != 0 {
		t.Errorf("clearUsageFields() did not clear IncludedUsage, got %v", price.IncludedUsage)
	}
	if price.UsageLimit != 0 {
		t.Errorf("clearUsageFields() did not clear UsageLimit, got %v", price.UsageLimit)
	}
}

func TestPrice_ConfigureUsageBilling(t *testing.T) {
	tests := []struct {
		name    string
		price   *Price
		input   CreatePriceInput
		wantErr bool
	}{
		{
			name:  "Valid usage configuration",
			price: &Price{},
			input: CreatePriceInput{
				HasUsage:         true,
				UsageType:        string(prices.UsageTypeMetered),
				UnitType:         string(prices.UnitTypeApiCalls),
				AggregationType:  string(prices.AggregationTypeSum),
				PercentageRate:   0,
				FixedFee:         100,
				OverageUnitPrice: 10,
				IncludedUsage:    1000,
				UsageLimit:       5000,
			},
			wantErr: false,
		},
		{
			name:  "Missing usage type",
			price: &Price{},
			input: CreatePriceInput{
				HasUsage:        true,
				UnitType:        string(prices.UnitTypeApiCalls),
				AggregationType: string(prices.AggregationTypeSum),
			},
			wantErr: true,
		},
		{
			name:  "Missing unit type",
			price: &Price{},
			input: CreatePriceInput{
				HasUsage:        true,
				UsageType:       string(prices.UsageTypeMetered),
				AggregationType: string(prices.AggregationTypeSum),
			},
			wantErr: true,
		},
		{
			name:  "Missing aggregation type",
			price: &Price{},
			input: CreatePriceInput{
				HasUsage:  true,
				UsageType: string(prices.UsageTypeMetered),
				UnitType:  string(prices.UnitTypeApiCalls),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.price.configureUsageBilling(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("configureUsageBilling() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !tt.price.HasUsage {
					t.Errorf("configureUsageBilling() did not set HasUsage to true")
				}
				if tt.price.UsageType != prices.UsageType(tt.input.UsageType) {
					t.Errorf("configureUsageBilling() did not set UsageType correctly, got %v, want %v",
						tt.price.UsageType, prices.UsageType(tt.input.UsageType))
				}
				if tt.price.UnitType != prices.UnitType(tt.input.UnitType) {
					t.Errorf("configureUsageBilling() did not set UnitType correctly, got %v, want %v",
						tt.price.UnitType, prices.UnitType(tt.input.UnitType))
				}
				if tt.price.AggregationType != prices.AggregationType(tt.input.AggregationType) {
					t.Errorf("configureUsageBilling() did not set AggregationType correctly, got %v, want %v",
						tt.price.AggregationType, prices.AggregationType(tt.input.AggregationType))
				}
			}
		})
	}
}
