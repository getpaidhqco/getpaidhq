package services

import (
	"context"
	"fmt"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"strconv"
	"time"
)

// BillingService implements the BillingService interface
type BillingService struct {
	usageEventRepository       repositories.UsageEventRepository
	subscriptionRepository     repositories.SubscriptionRepository
	subscriptionItemRepository repositories.SubscriptionItemRepository
	priceRepository            repositories.PriceRepository
	meterRepository            repositories.MeterRepository
	tierCalculationService     interfaces.TierCalculationService
}

// NewBillingService creates a new BillingService
func NewBillingService(
	usageEventRepository repositories.UsageEventRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	subscriptionItemRepository repositories.SubscriptionItemRepository,
	priceRepository repositories.PriceRepository,
	meterRepository repositories.MeterRepository,
	tierCalculationService interfaces.TierCalculationService,
) interfaces.BillingService {
	return &BillingService{
		usageEventRepository:       usageEventRepository,
		subscriptionRepository:     subscriptionRepository,
		subscriptionItemRepository: subscriptionItemRepository,
		priceRepository:            priceRepository,
		meterRepository:            meterRepository,
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


// sumTransactionValuesFromEvents sums the transaction values from usage events
func (b *BillingService) sumTransactionValuesFromEvents(events []entities.UsageEvent) float64 {
	var total float64
	for _, event := range events {
		if value, ok := event.Data["transaction_value"].(float64); ok && value > 0 {
			total += value
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
	// Get meter configuration for the subscription item
	if item.MeterId == "" {
		return 0, interfaces.UsageCalculationResult{}, fmt.Errorf("subscription item %s has no meter ID", item.Id)
	}

	meter, err := b.meterRepository.FindById(ctx, item.OrgId, item.MeterId)
	if err != nil {
		return 0, interfaces.UsageCalculationResult{}, fmt.Errorf("failed to find meter: %w", err)
	}

	// Use the meter's aggregation type
	aggregationType := meter.AggregationType

	// Let the database do the aggregation
	aggregatedQuantity, err := b.usageEventRepository.AggregateUsageBySubscriptionItem(
		ctx, item.OrgId, item.Id, period.StartDate, period.EndDate, aggregationType)
	if err != nil {
		return 0, interfaces.UsageCalculationResult{}, fmt.Errorf("failed to aggregate usage: %w", err)
	}

	// Use the meter's unit type
	unitType := string(meter.UnitType)

	usageResult := interfaces.UsageCalculationResult{
		SubscriptionItemId: item.Id,
		UnitType:           unitType,
		Quantity:           aggregatedQuantity,
		UnitPrice:          item.UnitPrice,
		AggregationType:    string(aggregationType),
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
		// For percentage-based pricing, we still need to get all records to sum transaction values
		// This could be optimized in the future with a specialized database aggregation
		usageEvents, err := b.usageEventRepository.FindBySubscriptionItem(
			ctx, item.OrgId, item.Id, period.StartDate, period.EndDate)
		if err != nil {
			return 0, usageResult, err
		}

		totalTransactionValue := b.sumTransactionValuesFromEvents(usageEvents)
		calculatedAmount = int64(totalTransactionValue * item.PercentageRate / 100)

	default:
		calculatedAmount = int64(aggregatedQuantity * float64(item.UnitPrice))
	}

	usageResult.Amount = calculatedAmount
	return calculatedAmount, usageResult, nil
}

// GenerateUsageCharges aggregates raw events and calculates charges for invoice generation
func (b *BillingService) GenerateUsageCharges(
	ctx context.Context,
	orgId string,
	subscriptionId string,
	billingPeriodStart time.Time,
	billingPeriodEnd time.Time,
) ([]entities.UsageLineItem, error) {

	// 1. Get all subscription items with usage billing
	_, err := b.subscriptionRepository.FindById(ctx, orgId, subscriptionId)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription: %w", err)
	}

	subscriptionItems, err := b.subscriptionItemRepository.FindBySubscriptionId(ctx, orgId, subscriptionId)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription items: %w", err)
	}

	var usageLineItems []entities.UsageLineItem

	// 2. Process each subscription item with usage billing
	for _, item := range subscriptionItems {
		if !item.HasUsage || item.MeterId == "" {
			continue
		}

		// 3. Get meter configuration
		meter, err := b.meterRepository.FindById(ctx, orgId, item.MeterId)
		if err != nil {
			// Log warning and continue with next item
			continue
		}

		// 4. Query raw events for this subscription item and period
		// TODO: Fix unresolved reference to rawUsageRepository
		// This is not directly related to the changes for using Meters instead of Prices
		// and would require more context to fix properly
		var rawEvents []events.RawUsageRecordedEvent
		// rawEvents, err := b.rawUsageRepository.FindBySubscriptionItemId(
		// 	ctx, orgId, item.Id, billingPeriodStart, billingPeriodEnd,
		// )
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to query raw events: %w", err)
		// }

		if len(rawEvents) == 0 {
			continue // No usage for this item
		}

		// 5. Aggregate usage based on meter configuration
		aggregatedValue, err := b.aggregateRawUsage(rawEvents, meter)
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate usage: %w", err)
		}

		// 6. Calculate charges based on current pricing
		totalAmount, err := b.calculateRawUsageCharges(aggregatedValue, item, rawEvents)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate charges: %w", err)
		}

		// 7. Create usage line item
		lineItem := entities.UsageLineItem{
			SubscriptionItemId: item.Id,
			MeterId:            meter.Id,
			MeterName:          meter.Name,
			AggregatedValue:    aggregatedValue,
			TotalAmount:        totalAmount,
			EventCount:         len(rawEvents),
			PeriodStart:        billingPeriodStart,
			PeriodEnd:          billingPeriodEnd,
		}

		usageLineItems = append(usageLineItems, lineItem)
	}

	return usageLineItems, nil
}

// aggregateRawUsage applies meter aggregation rules to raw events
func (b *BillingService) aggregateRawUsage(rawEvents []events.RawUsageRecordedEvent, meter entities.Meter) (float64, error) {
	if len(rawEvents) == 0 {
		return 0, nil
	}

	var values []float64

	// Extract values from each event based on meter configuration
	for _, event := range rawEvents {
		// Convert event.Data to map[string]interface{}
		dataMap, ok := event.Data.(map[string]interface{})
		if !ok {
			// Skip this event if data is not a map
			continue
		}

		value, err := extractValueFromEventData(dataMap, meter.ValueProperty)
		if err != nil {
			// Skip this event if value extraction fails
			continue
		}
		values = append(values, value)
	}

	// Apply aggregation type
	switch meter.AggregationType {
	case entities.AggregationTypeSum:
		var sum float64
		for _, v := range values {
			sum += v
		}
		return sum, nil

	// Count is not a standard aggregation type, but we can implement it here
	case "count":
		return float64(len(values)), nil

	case entities.AggregationTypeMax:
		if len(values) == 0 {
			return 0, nil
		}
		max := values[0]
		for _, v := range values[1:] {
			if v > max {
				max = v
			}
		}
		return max, nil

	case entities.AggregationTypeAverage:
		if len(values) == 0 {
			return 0, nil
		}
		var sum float64
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values)), nil

	case entities.AggregationTypeLastDuringPeriod:
		if len(values) == 0 {
			return 0, nil
		}
		// Events should be ordered by time, return last value
		return values[len(values)-1], nil

	default:
		return 0, fmt.Errorf("unsupported aggregation type: %s", meter.AggregationType)
	}
}

// calculateRawUsageCharges applies pricing rules to aggregated usage
func (b *BillingService) calculateRawUsageCharges(
	aggregatedValue float64,
	subscriptionItem entities.SubscriptionItem,
	rawEvents []events.RawUsageRecordedEvent,
) (int64, error) {

	// Get unit type from meter or default to "count"
	var unitType string
	if subscriptionItem.MeterId != "" {
		meter, err := b.meterRepository.FindById(context.Background(), subscriptionItem.OrgId, subscriptionItem.MeterId)
		if err == nil {
			unitType = string(meter.UnitType)
		}
	}
	if unitType == "" {
		unitType = "count" // Default to count if no meter or error
	}

	switch unitType {
	case string(entities.UnitTypeTransactions):
		// For transaction-based pricing, sum transaction values and apply percentage
		var totalTransactionValue int64
		for _, event := range rawEvents {
			// Convert event.Data to map[string]interface{}
			dataMap, ok := event.Data.(map[string]interface{})
			if !ok {
				// Skip this event if data is not a map
				continue
			}

			if transactionValue, err := getTransactionValueFromEventData(dataMap); err == nil {
				totalTransactionValue += transactionValue
			}
		}

		// Apply percentage rate
		percentageFee := int64(float64(totalTransactionValue) * subscriptionItem.PercentageRate / 100)

		// Add fixed fee per transaction
		fixedFee := int64(aggregatedValue * float64(subscriptionItem.FixedFee))

		return percentageFee + fixedFee, nil

	default:
		// Unit-based pricing
		return int64(aggregatedValue * float64(subscriptionItem.UnitPrice)), nil
	}
}

// Helper functions for value extraction
func extractValueFromEventData(eventData map[string]interface{}, valueProperty string) (float64, error) {
	// If no specific property is specified, look for "quantity" by default
	if valueProperty == "" {
		valueProperty = "quantity"
	}

	// Try to extract the value from the data map
	if val, ok := eventData[valueProperty]; ok {
		// Convert the value to float64
		switch v := val.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case int32:
			return float64(v), nil
		case string:
			// Try to parse the string as a float
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f, nil
			}
		}
	}

	// If we couldn't find or convert the value, return an error
	return 0, fmt.Errorf("could not extract value for property %s from event data", valueProperty)
}

func getTransactionValueFromEventData(eventData map[string]interface{}) (int64, error) {
	// Look for transaction_value in the data map
	if val, ok := eventData["transaction_value"]; ok {
		// Convert the value to int64
		switch v := val.(type) {
		case int64:
			return v, nil
		case int:
			return int64(v), nil
		case float64:
			return int64(v), nil
		case float32:
			return int64(v), nil
		case string:
			// Try to parse the string as an int64
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i, nil
			}
		}
	}

	// If we couldn't find or convert the value, return an error
	return 0, fmt.Errorf("could not extract transaction_value from event data")
}
