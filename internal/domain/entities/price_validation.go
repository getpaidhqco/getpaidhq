package entities

import (
    "errors"
    "payloop/internal/domain/entities/prices"
)

// Validate ensures the Price entity has consistent configuration
func (p *Price) Validate() error {
    // Validate based on category
    switch p.Category {
    case prices.PriceCategoryUsage:
        if !p.HasUsage {
            return errors.New("usage category requires has_usage to be true")
        }
        if p.UsageType == "" || p.UnitType == "" || p.AggregationType == "" {
            return errors.New("usage category requires usage_type, unit_type, and aggregation_type")
        }

    case prices.PriceCategoryHybrid:
        if !p.HasUsage {
            return errors.New("hybrid category requires has_usage to be true")
        }
        if p.UnitPrice == 0 && p.OverageUnitPrice == 0 {
            return errors.New("hybrid category requires either base price or overage price")
        }

    case prices.PriceCategorySubscription:
        if p.BillingInterval == prices.BillingIntervalNone && p.HasUsage {
            return errors.New("subscription with usage requires billing interval")
        }
    }

    // Validate percentage-based pricing
    if p.PercentageRate > 0 && p.UnitType != "transactions" && p.UnitType != "cents" && p.UnitType != "dollars" {
        return errors.New("percentage rate only valid for transaction-based units")
    }

    return nil
}