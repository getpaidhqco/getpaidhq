package services

import (
	"context"
	"fmt"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"time"
)

// BillingService implements the BillingService interface
type BillingService struct {
	usageRecordRepository      repositories.UsageRecordRepository
	subscriptionItemRepository repositories.SubscriptionItemRepository
	priceRepository            repositories.PriceRepository
	tierCalculationService     interfaces.TierCalculationService
}

// NewBillingService creates a new BillingService
func NewBillingService(
	usageRecordRepository repositories.UsageRecordRepository,
	subscriptionItemRepository repositories.SubscriptionItemRepository,
	priceRepository repositories.PriceRepository,
	tierCalculationService interfaces.TierCalculationService,
) interfaces.BillingService {
	return &BillingService{
		usageRecordRepository:      usageRecordRepository,
		subscriptionItemRepository: subscriptionItemRepository,
		priceRepository:            priceRepository,
		tierCalculationService:     tierCalculationService,
	}
}

// CalculateBillingAmount calculates the total billing amount for a subscription
func (b *BillingService) CalculateBillingAmount(ctx context.Context, subscription entities.Subscription) (interfaces.BillingCalculation, error) {
	var calculation interfaces.BillingCalculation
	calculation.Currency = subscription.Currency

	// Get current billing period
	period := b.getCurrentBillingPeriod(subscription)

	// Get subscription items from repository if not loaded
	var items []entities.SubscriptionItem
	if len(subscription.Items) > 0 {
		items = subscription.Items
	} else {
		var err error
		items, err = b.subscriptionItemRepository.FindBySubscriptionId(ctx, subscription.OrgId, subscription.Id)
		if err != nil {
			return interfaces.BillingCalculation{}, err
		}
	}

	// For legacy subscriptions without items, use the subscription amount
	if len(items) == 0 {
		calculation.BaseAmount = subscription.Amount
		calculation.TotalAmount = subscription.Amount
		calculation.ItemBreakdown = []interfaces.BillingItemBreakdown{{
			SubscriptionItemId: subscription.Id,
			Description:        "Legacy subscription",
			PriceCategory:      "subscription",
			Amount:             subscription.Amount,
		}}
		return calculation, nil
	}

	// Calculate amounts for each subscription item
	for _, item := range items {
		itemAmount, usageResult, err := b.calculateItemAmount(ctx, item, period)
		if err != nil {
			return interfaces.BillingCalculation{}, err
		}

		priceCategory := b.getPriceCategory(item)

		// Create breakdown entry
		breakdown := interfaces.BillingItemBreakdown{
			SubscriptionItemId: item.Id,
			Description:        item.Description,
			PriceCategory:      priceCategory,
			Amount:             itemAmount,
		}
		calculation.ItemBreakdown = append(calculation.ItemBreakdown, breakdown)

		// Add usage calculation details if applicable
		if usageResult.Quantity > 0 {
			calculation.UsageBreakdown = append(calculation.UsageBreakdown, usageResult)
		}

		// Add to appropriate totals based on price category
		switch priceCategory {
		case "subscription":
			calculation.BaseAmount += itemAmount
		case "usage":
			calculation.UsageAmount += itemAmount
		case "hybrid":
			calculation.BaseAmount += item.Amount
			overageAmount := itemAmount - item.Amount
			if overageAmount > 0 {
				calculation.UsageAmount += overageAmount
			}
		default:
			calculation.BaseAmount += itemAmount
		}
	}

	// Calculate proration adjustments
	prorationAmount, err := b.CalculateProrationAdjustments(ctx, subscription)
	if err != nil {
		return interfaces.BillingCalculation{}, err
	}
	calculation.ProrationAmount = prorationAmount

	// Calculate final total
	calculation.TotalAmount = calculation.BaseAmount + calculation.UsageAmount + calculation.ProrationAmount

	return calculation, nil
}

// getCurrentBillingPeriod returns the current billing period for a subscription
func (b *BillingService) getCurrentBillingPeriod(subscription entities.Subscription) interfaces.BillingPeriod {
	// Use provided period dates if available
	if !subscription.CurrentPeriodStart.IsZero() && !subscription.CurrentPeriodEnd.IsZero() {
		return interfaces.BillingPeriod{
			StartDate: subscription.CurrentPeriodStart,
			EndDate:   subscription.CurrentPeriodEnd,
		}
	}

	// Calculate based on billing interval
	now := time.Now().UTC()
	startDate := subscription.CreatedAt

	// Find the current period based on billing interval
	switch subscription.BillingInterval {
	case "monthly":
		monthsSinceStart := int(now.Sub(startDate).Hours() / 24 / 30)
		startDate = startDate.AddDate(0, monthsSinceStart, 0)
		endDate := startDate.AddDate(0, 1, 0)
		return interfaces.BillingPeriod{
			StartDate: startDate,
			EndDate:   endDate,
		}
	case "yearly":
		yearsSinceStart := int(now.Sub(startDate).Hours() / 24 / 365)
		startDate = startDate.AddDate(yearsSinceStart, 0, 0)
		endDate := startDate.AddDate(1, 0, 0)
		return interfaces.BillingPeriod{
			StartDate: startDate,
			EndDate:   endDate,
		}
	case "weekly":
		weeksSinceStart := int(now.Sub(startDate).Hours() / 24 / 7)
		startDate = startDate.AddDate(0, 0, weeksSinceStart*7)
		endDate := startDate.AddDate(0, 0, 7)
		return interfaces.BillingPeriod{
			StartDate: startDate,
			EndDate:   endDate,
		}
	default:
		// Default to monthly
		endDate := startDate.AddDate(0, 1, 0)
		return interfaces.BillingPeriod{
			StartDate: startDate,
			EndDate:   endDate,
		}
	}
}

// CalculateTraditionalAmount calculates the amount for a traditional subscription
func (b *BillingService) CalculateTraditionalAmount(ctx context.Context, subscription entities.Subscription) (int64, error) {
	// For simple subscriptions without items, use the subscription amount
	if len(subscription.Items) == 0 {
		return subscription.Amount, nil
	}

	// Calculate total from subscription items
	var totalAmount int64
	for _, item := range subscription.Items {
		if !item.HasUsage {
			totalAmount += item.Amount
		}
	}

	return totalAmount, nil
}

// CalculateUsageAmount calculates the amount for usage-based billing
func (b *BillingService) CalculateUsageAmount(ctx context.Context, subscription entities.Subscription, period interfaces.BillingPeriod) (int64, error) {
	var totalAmount int64
	for _, item := range subscription.Items {
		if item.HasUsage {
			itemAmount, _, err := b.calculateItemAmount(ctx, item, period)
			if err != nil {
				return 0, err
			}
			totalAmount += itemAmount
		}
	}

	return totalAmount, nil
}

// CalculateHybridAmount calculates the amount for hybrid billing (base + usage)
func (b *BillingService) CalculateHybridAmount(ctx context.Context, subscription entities.Subscription, period interfaces.BillingPeriod) (int64, error) {
	// Calculate base amount
	baseAmount, err := b.CalculateTraditionalAmount(ctx, subscription)
	if err != nil {
		return 0, err
	}

	// Calculate usage amount
	usageAmount, err := b.CalculateUsageAmount(ctx, subscription, period)
	if err != nil {
		return 0, err
	}

	return baseAmount + usageAmount, nil
}

// CalculateProrationAdjustments calculates proration adjustments for a subscription
func (b *BillingService) CalculateProrationAdjustments(ctx context.Context, subscription entities.Subscription) (int64, error) {
	// Check if subscription has pending proration metadata
	if subscription.Metadata == nil {
		return 0, nil
	}

	prorationAmount := int64(0)

	// Check for billing anchor change proration
	if amountStr, ok := subscription.Metadata["pending_proration_amount"]; ok {
		var amount float64
		_, err := fmt.Sscanf(amountStr, "%f", &amount)
		if err == nil {
			prorationAmount = int64(amount)
			// Note: In production, you'd clear this metadata after processing
		}
	}

	// Check for plan change proration
	if planChangeId, ok := subscription.Metadata["pending_plan_change_id"]; ok && planChangeId != "" {
		// In production, fetch the plan change details and calculate proration
		// For now, return any stored proration amount
		if amountStr, ok := subscription.Metadata["plan_change_proration"]; ok {
			var amount float64
			_, err := fmt.Sscanf(amountStr, "%f", &amount)
			if err == nil {
				prorationAmount += int64(amount)
			}
		}
	}

	return prorationAmount, nil
}

// getPriceCategory determines the price category for a subscription item
func (b *BillingService) getPriceCategory(item entities.SubscriptionItem) string {
	// Check if item has price category in metadata
	if category, ok := item.Metadata["price_category"]; ok {
		return category
	}

	// Determine based on item properties
	if item.HasUsage && item.Amount > 0 {
		return "hybrid"
	} else if item.HasUsage {
		return "usage"
	}

	return "subscription"
}

// Helper functions to get values from metadata or defaults
func getIncludedUsage(item entities.SubscriptionItem) int64 {
	if val, ok := item.Metadata["included_usage"]; ok {
		// Try to parse the value as an integer
		var includedUsage int64
		_, err := fmt.Sscanf(val, "%d", &includedUsage)
		if err == nil {
			return includedUsage
		}
	}
	return 0
}

func getOverageUnitPrice(item entities.SubscriptionItem) int64 {
	if val, ok := item.Metadata["overage_unit_price"]; ok {
		// Try to parse the value as an integer
		var overageUnitPrice int64
		_, err := fmt.Sscanf(val, "%d", &overageUnitPrice)
		if err == nil {
			return overageUnitPrice
		}
	}
	// Default to regular unit price if not specified
	return item.UnitPrice
}

func getPricingScheme(item entities.SubscriptionItem) string {
	if scheme, ok := item.Metadata["pricing_scheme"]; ok {
		return scheme
	}
	return "fixed"
}

// aggregateUsage aggregates usage records based on the specified aggregation type
func (b *BillingService) aggregateUsage(records []entities.UsageRecord, aggregationType entities.AggregationType) float64 {
	if len(records) == 0 {
		return 0
	}

	switch aggregationType {
	case entities.AggregationTypeSum:
		var total float64
		for _, record := range records {
			total += record.Quantity
		}
		return total

	case entities.AggregationTypeMax:
		max := records[0].Quantity
		for _, record := range records[1:] {
			if record.Quantity > max {
				max = record.Quantity
			}
		}
		return max

	case entities.AggregationTypeAverage:
		var total float64
		for _, record := range records {
			total += record.Quantity
		}
		return total / float64(len(records))

	case entities.AggregationTypeLastDuringPeriod:
		latest := records[0]
		for _, record := range records[1:] {
			if record.CreatedAt.After(latest.CreatedAt) {
				latest = record
			}
		}
		return latest.Quantity

	default:
		// Default to sum
		var total float64
		for _, record := range records {
			total += record.Quantity
		}
		return total
	}
}

// sumTransactionValues sums the transaction values from usage records
func (b *BillingService) sumTransactionValues(records []entities.UsageRecord) float64 {
	var total float64
	for _, record := range records {
		if record.TransactionValue > 0 {
			total += float64(record.TransactionValue)
		}
	}
	return total
}

// calculateItemAmount calculates the amount for a subscription item with price category support
func (b *BillingService) calculateItemAmount(ctx context.Context, item entities.SubscriptionItem, period interfaces.BillingPeriod) (int64, interfaces.UsageCalculationResult, error) {
	// Determine price category from item metadata or default
	priceCategory := b.getPriceCategory(item)

	switch priceCategory {
	case "subscription":
		return item.Amount, interfaces.UsageCalculationResult{}, nil

	case "usage":
		return b.calculateUsageItemAmount(ctx, item, period)

	case "hybrid":
		baseAmount := item.Amount
		usageAmount, usageResult, err := b.calculateUsageItemAmount(ctx, item, period)
		if err != nil {
			return 0, interfaces.UsageCalculationResult{}, err
		}

		// For hybrid, check if usage exceeds included amount
		includedUsage := getIncludedUsage(item)
		if includedUsage > 0 && usageResult.Quantity <= float64(includedUsage) {
			// Usage within included amount, no additional charge
			return baseAmount, usageResult, nil
		}

		// Calculate overage
		if includedUsage > 0 {
			usageResult.Quantity = usageResult.Quantity - float64(includedUsage)
			overageUnitPrice := getOverageUnitPrice(item)
			usageAmount = int64(usageResult.Quantity * float64(overageUnitPrice))
			usageResult.Amount = usageAmount
		}

		return baseAmount + usageAmount, usageResult, nil

	default:
		return item.Amount, interfaces.UsageCalculationResult{}, nil
	}
}

// calculateUsageItemAmount calculates the amount for a usage-based subscription item
func (b *BillingService) calculateUsageItemAmount(ctx context.Context, item entities.SubscriptionItem, period interfaces.BillingPeriod) (int64, interfaces.UsageCalculationResult, error) {
	// Get usage records for the period
	usageRecords, err := b.usageRecordRepository.FindBySubscriptionItem(ctx, item.OrgId, item.Id, period.StartDate, period.EndDate)
	if err != nil {
		return 0, interfaces.UsageCalculationResult{}, err
	}

	// Aggregate usage based on aggregation type
	aggregatedQuantity := b.aggregateUsage(usageRecords, item.AggregationType)

	usageResult := interfaces.UsageCalculationResult{
		SubscriptionItemId: item.Id,
		UnitType:           string(item.UnitType),
		Quantity:           aggregatedQuantity,
		UnitPrice:          item.UnitPrice,
		AggregationType:    string(item.AggregationType),
	}

	// Calculate amount based on pricing scheme
	var calculatedAmount int64

	pricingScheme := getPricingScheme(item)

	switch pricingScheme {
	case "fixed":
		calculatedAmount = int64(aggregatedQuantity * float64(item.UnitPrice))

	case "tiered", "volume", "graduated":
		// Get price for tier calculations
		price, err := b.priceRepository.FindById(ctx, item.OrgId, item.PriceId)
		if err != nil {
			return 0, usageResult, err
		}

		tierResult, err := b.tierCalculationService.CalculateTieredAmount(ctx, int(aggregatedQuantity), price)
		if err != nil {
			return 0, usageResult, err
		}

		calculatedAmount = tierResult.TotalAmount

	case "percentage":
		// For transaction-based fees
		totalTransactionValue := b.sumTransactionValues(usageRecords)
		calculatedAmount = int64(totalTransactionValue * item.PercentageRate / 100)

	default:
		calculatedAmount = int64(aggregatedQuantity * float64(item.UnitPrice))
	}

	usageResult.Amount = calculatedAmount
	return calculatedAmount, usageResult, nil
}
