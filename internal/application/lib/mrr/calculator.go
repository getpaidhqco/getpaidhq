package mrr

import (
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
)

// Calculator handles MRR calculations for subscriptions
type Calculator struct{}

// NewCalculator creates a new MRR calculator instance
func NewCalculator() Calculator {
	return Calculator{}
}

// CalculateCustomerMrr calculates the MRR for all active subscriptions of a customer
func (c Calculator) CalculateCustomerMrr(customerId string, subscriptions []entities.Subscription) dto.CustomerMrrData {
	var totalMrr int64
	var currency string
	var breakdown []dto.MrrBreakdownData
	
	for _, subscription := range subscriptions {
		// Skip inactive subscriptions
		if !c.isActiveSubscription(subscription) {
			continue
		}
		
		// Set currency from first active subscription
		if currency == "" {
			currency = subscription.Currency
		}
		
		// Calculate MRR for this subscription
		subscriptionMrr := c.calculateSubscriptionMrr(subscription)
		totalMrr += subscriptionMrr.MonthlyAmount
		
		breakdown = append(breakdown, subscriptionMrr)
	}
	
	return dto.CustomerMrrData{
		CustomerId:             customerId,
		TotalMrr:               totalMrr,
		Currency:               currency,
		Breakdown:              breakdown,
		ProjectedAnnualRevenue: totalMrr * 12,
	}
}

// calculateSubscriptionMrr calculates MRR for a single subscription
func (c Calculator) calculateSubscriptionMrr(subscription entities.Subscription) dto.MrrBreakdownData {
	// For backward compatibility, handle simple subscriptions with Amount field
	if subscription.Amount > 0 {
		monthlyAmount := c.normalizeToMonthly(subscription.Amount, subscription.BillingInterval, subscription.BillingIntervalQty)
		
		return dto.MrrBreakdownData{
			SubscriptionId:    subscription.Id,
			ProductName:       "Subscription", // Default name for legacy subscriptions
			MonthlyAmount:     monthlyAmount,
			BillingInterval:   string(subscription.BillingInterval),
			NormalizedMonthly: monthlyAmount,
			NextBilling:       subscription.RenewsAt,
		}
	}
	
	// For modern subscriptions with items
	var totalMonthlyAmount int64
	productName := "Multi-item Subscription" // Default for multiple items
	
	if len(subscription.Items) == 1 {
		productName = subscription.Items[0].Name
	}
	
	for _, item := range subscription.Items {
		// Only include fixed amounts, exclude pure usage-based components
		if item.Amount > 0 || item.FixedFee > 0 {
			itemAmount := item.Amount
			if itemAmount == 0 && item.FixedFee > 0 {
				itemAmount = item.FixedFee * int64(item.Quantity)
			}
			
			// Note: We use subscription-level billing interval as items inherit it
			monthlyAmount := c.normalizeToMonthly(itemAmount, subscription.BillingInterval, subscription.BillingIntervalQty)
			totalMonthlyAmount += monthlyAmount
		}
	}
	
	return dto.MrrBreakdownData{
		SubscriptionId:    subscription.Id,
		ProductName:       productName,
		MonthlyAmount:     totalMonthlyAmount,
		BillingInterval:   string(subscription.BillingInterval),
		NormalizedMonthly: totalMonthlyAmount,
		NextBilling:       subscription.RenewsAt,
	}
}

// normalizeToMonthly converts any billing interval to monthly amount
func (c Calculator) normalizeToMonthly(amount int64, interval prices.BillingInterval, intervalQty int) int64 {
	if amount == 0 {
		return 0
	}
	
	// Handle quantity (e.g., every 3 months)
	if intervalQty == 0 {
		intervalQty = 1
	}
	
	switch interval {
	case prices.BillingIntervalMonth:
		// Already monthly, just adjust for quantity
		return amount / int64(intervalQty)
		
	case prices.BillingIntervalYear:
		// Annual to monthly: divide by (12 * intervalQty)
		return amount / (12 * int64(intervalQty))
		
	case prices.BillingIntervalWeek:
		// Weekly to monthly: multiply by 4.33 (52 weeks / 12 months), then divide by qty
		return (amount * 433) / (100 * int64(intervalQty))
		
	case prices.BillingIntervalDay:
		// Daily to monthly: multiply by 30.44 (365.25 days / 12 months), then divide by qty
		return (amount * 3044) / (100 * int64(intervalQty))
		
	case prices.BillingIntervalHour:
		// Hourly to monthly: 24 hours * 30.44 days = 730.56 hours per month
		return (amount * 73056) / (100 * int64(intervalQty))
		
	case prices.BillingIntervalMinute:
		// Minute to monthly: 60 * 730.56 = 43,833.6 minutes per month
		return (amount * 4383360) / (100 * int64(intervalQty))
		
	case prices.BillingIntervalSecond:
		// Second to monthly: 60 * 43,833.6 = 2,630,016 seconds per month
		return (amount * 263001600) / (100 * int64(intervalQty))
		
	default:
		// For unknown intervals or "none", return the amount as-is
		return amount
	}
}

// isActiveSubscription determines if a subscription should be included in MRR calculations
func (c Calculator) isActiveSubscription(subscription entities.Subscription) bool {
	switch subscription.Status {
	case "active", "trialing":
		return true
	case "past_due":
		// Include past due subscriptions as they're still expected to generate revenue
		return true
	default:
		return false
	}
}