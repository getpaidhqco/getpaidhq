package entities

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/lib"
	"time"
)

type CreateSubscriptionInput struct {
	OrgId string `json:"org_id"`

	PaymentMethodId string `json:"payment_method_id" binding:"required"`
	Activate        bool   `json:"activate"`

	Amount   int64  `json:"amount"  binding:"required"`
	Currency string `json:"currency"  binding:"required"`

	BillingInterval    prices.BillingInterval `json:"billing_interval"  binding:"required"`
	BillingIntervalQty int                    `json:"billing_interval_qty"  binding:"required"`
	Cycles             int                    `json:"cycles"`

	TrialInterval    prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty int                    `json:"trial_interval_qty"`

	Metadata map[string]string `json:"metadata"`
}

type SubscriptionStatus string

const (
	SubscriptionStatusTrial  SubscriptionStatus = "trial"
	SubscriptionStatusActive SubscriptionStatus = "active"

	// SubscriptionStatusPastDue The initial schedule charge failed, so the subscription is in a retry workflow
	// that will attempt to charge the customer again.  The retry workflow can't be
	// longer than the subscription period, so if the retry fails, the subscription
	// will be marked as past_due
	SubscriptionStatusPastDue SubscriptionStatus = "past_due"

	SubscriptionStatusNonRenewing SubscriptionStatus = "non_renewing"
	SubscriptionStatusPaused      SubscriptionStatus = "paused"
	SubscriptionStatusUnpaid      SubscriptionStatus = "unpaid"
	SubscriptionStatusCancelled   SubscriptionStatus = "cancelled"
	SubscriptionStatusPending     SubscriptionStatus = "pending"
	SubscriptionStatusExpired     SubscriptionStatus = "expired"
	SubscriptionStatusCompleted   SubscriptionStatus = "completed"
	SubscriptionStatusError       SubscriptionStatus = "error"
)

type Subscription struct {
	OrgId           string             `json:"org_id"`
	Id              string             `json:"id"`
	PspId           common.Gateway     `json:"psp_id"`
	OrderId         string             `json:"order_id"`
	OrderItemId     string             `json:"order_item_id"`
	OrderItem       OrderItem          `json:"-"`
	CustomerId      string             `json:"customer_id"`
	Customer        Customer           `json:"-"`
	Status          SubscriptionStatus `json:"status"`
	PaymentMethodId string             `json:"payment_method_id,omitempty"`

	// Optional amount and currency for backward compatibility
	// For simple subscriptions, use these fields
	// For multi-item subscriptions, these will be null and calculated from items
	Amount   int64  `json:"amount,omitempty"`
	Currency string `json:"currency,omitempty"`

	// StartDate is the date when the subscription was activated.
	// It doesn't include the trial period, if any.
	StartDate          time.Time              `json:"start_date"`
	EndDate            time.Time              `json:"end_date,omitempty,omitzero"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	Cycles             int                    `json:"cycles"`
	BillingAnchor      int                    `json:"billing_anchor"`

	// TrialEndsAt is the date when the trial period ends, calculated relative to StartDate.
	TrialEndsAt time.Time `json:"trial_ends_at,omitempty,omitzero"`

	// CancelAt is the date when the subscription will be cancelled, this is used when the subscription is a
	// payment plan.
	CancelAt   time.Time `json:"cancel_at,omitempty,omitzero"`
	EndsAt     time.Time `json:"ends_at,omitempty,omitzero"`
	LastCharge time.Time `json:"last_charge"`

	// RenewsAt is the date when the subscription will be charged next. This is always the
	// date based on the billing anchor and interval. Retry dates are not included in this date.
	RenewsAt time.Time `json:"renews_at"`

	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`

	// DunningActive indicates if the subscription is currently in an active dunning process
	DunningActive bool `json:"dunning_active"`

	// ActiveDunningCampaignId is the ID of the active dunning campaign for this subscription
	ActiveDunningCampaignId string `json:"active_dunning_campaign_id,omitempty"`

	Metadata        map[string]string `json:"metadata"`
	CyclesProcessed int               `json:"cycles_processed"`
	TotalRevenue    int64             `json:"total_revenue"`
	CancelledAt     time.Time         `json:"cancelled_at,omitempty,omitzero"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`

	// Subscription items
	Items []SubscriptionItem `json:"items,omitempty"`
}

type ProrationDetails struct {
	CreditAmount       int       `json:"credit_amount"`
	DaysCredited       int       `json:"days_credited"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	OldBillingAnchor   int       `json:"old_billing_anchor,omitempty"`
	NewBillingAnchor   int       `json:"new_billing_anchor,omitempty"`
	NewPeriodStart     time.Time `json:"new_period_start,omitempty"`
	NewPeriodEnd       time.Time `json:"new_period_end,omitempty"`
}

// CalculateProrationDetails calculates proration details based on the proration mode
// prorationMode: "none" or "credit_unused"
// referenceDate: the date to calculate proration from (usually the current date)
// oldBillingAnchor: the old billing anchor day (optional)
// newBillingAnchor: the new billing anchor day (optional)
// newPeriodStart: the start date of the new billing period (optional)
// newPeriodEnd: the end date of the new billing period (optional)
func (s *Subscription) CalculateProrationDetails(
	prorationMode string,
	referenceDate time.Time,
	oldBillingAnchor, newBillingAnchor int,
	newPeriodStart, newPeriodEnd time.Time,
) ProrationDetails {
	details := ProrationDetails{
		CreditAmount:       0,
		DaysCredited:       0,
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		OldBillingAnchor:   oldBillingAnchor,
		NewBillingAnchor:   newBillingAnchor,
		NewPeriodStart:     newPeriodStart,
		NewPeriodEnd:       newPeriodEnd,
	}

	// If proration mode is none, return zero credit
	if prorationMode == "none" {
		return details
	}

	// If proration mode is credit_unused, calculate the credit amount
	if prorationMode == "credit_unused" {
		// Calculate total days in the billing period
		totalDays := int(s.CurrentPeriodEnd.Sub(s.CurrentPeriodStart).Hours() / 24)
		if totalDays <= 0 {
			return details
		}

		// Calculate days remaining in the billing period from the reference date
		daysRemaining := int(s.CurrentPeriodEnd.Sub(referenceDate).Hours() / 24)
		if daysRemaining <= 0 {
			return details
		}

		// Get the total amount to prorate
		totalAmount := s.GetTotalAmount()

		// Calculate the credit amount based on the proportion of days remaining
		creditAmount := int(float64(totalAmount) * float64(daysRemaining) / float64(totalDays))

		details.CreditAmount = creditAmount
		details.DaysCredited = daysRemaining
	}

	return details
}

// IsRunning checks if the subscription is in an active or trial state.
func (s *Subscription) IsRunning() bool {
	return s.Status == SubscriptionStatusActive ||
		s.Status == SubscriptionStatusTrial
}

// GetNextChargeDate returns the next charge date for the subscription,
// which is RenewsAt if the subscription is in Active state.
func (s *Subscription) GetNextChargeDate() time.Time {
	if s.Status == SubscriptionStatusActive {
		return s.RenewsAt
	}

	return s.RenewsAt
}

// SetMetadata merges the existing metadata with the specified values.
func (s *Subscription) SetMetadata(meta map[string]string) *Subscription {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	for key, value := range meta {
		s.Metadata[key] = value
	}
	return s
}

// AddItem adds a new subscription item to the subscription
func (s *Subscription) AddItem(item SubscriptionItem) *Subscription {
	if s.Items == nil {
		s.Items = []SubscriptionItem{}
	}
	s.Items = append(s.Items, item)
	s.UpdatedAt = time.Now().UTC()
	return s
}

// RemoveItem removes a subscription item from the subscription
func (s *Subscription) RemoveItem(itemId string) *Subscription {
	if s.Items == nil {
		return s
	}

	var newItems []SubscriptionItem
	for _, item := range s.Items {
		if item.Id != itemId {
			newItems = append(newItems, item)
		}
	}

	s.Items = newItems
	s.UpdatedAt = time.Now().UTC()
	return s
}

// FindItem finds a subscription item by ID
func (s *Subscription) FindItem(itemId string) *SubscriptionItem {
	if s.Items == nil {
		return nil
	}

	for i, item := range s.Items {
		if item.Id == itemId {
			return &s.Items[i]
		}
	}

	return nil
}

// GetActiveItems returns all active subscription items
func (s *Subscription) GetActiveItems() []SubscriptionItem {
	if s.Items == nil {
		return []SubscriptionItem{}
	}

	var activeItems []SubscriptionItem
	for _, item := range s.Items {
		if item.Status == SubscriptionItemStatusActive {
			activeItems = append(activeItems, item)
		}
	}

	return activeItems
}

// GetTotalAmount calculates the total fixed amount for all active subscription items
func (s *Subscription) GetTotalAmount() int64 {
	if s.Items == nil || len(s.Items) == 0 {
		return 0
	}

	var total int64
	for _, item := range s.GetActiveItems() {
		if item.Amount > 0 {
			total += item.Amount * int64(item.Quantity)
		}
	}

	return total
}

// CalculateNextBillingDate calculates and returns the next billing date for a subscription
// based on the StartDate, BillingInterval, BillingIntervalQty and CyclesProcessed
// It can't use LastCharge as the base date because of retries - LastCharge could be in middle of the
// billing cycle if a retry policy is used.
//
// If the subscription has not started yet, it returns the StartDate
// If the subscription has started but has not been charged yet, it returns the StartDate

func (s *Subscription) CalculateNextBillingDate() time.Time {
	if s.BillingInterval == "" || s.BillingIntervalQty <= 0 {
		return time.Time{}
	}
	var nextBillingDate time.Time
	if s.LastCharge.IsZero() && s.CyclesProcessed == 0 {
		// new subscription, not charged yet
		return s.StartDate
	}

	//
	base := s.CurrentPeriodEnd

	switch s.BillingInterval {
	case "minute":
		nextBillingDate = base.Add(time.Minute * time.Duration(s.BillingIntervalQty))
	case "hour":
		nextBillingDate = base.Add(time.Hour * time.Duration(s.BillingIntervalQty))
	case "day":
		nextBillingDate = base.AddDate(0, 0, s.BillingIntervalQty)
	case "week":
		nextBillingDate = base.AddDate(0, 0, s.BillingIntervalQty*7)
	case "month":
		nextBillingDate = base.AddDate(0, s.BillingIntervalQty, 0)
	case "year":
		nextBillingDate = base.AddDate(s.BillingIntervalQty, 0, 0)
	}

	return nextBillingDate
}

// AddBillingInterval adds the billing interval to the base date and returns the new date
func (s *Subscription) AddBillingInterval(base time.Time) time.Time {
	var rsp time.Time
	switch s.BillingInterval {
	case "minute":
		rsp = base.Add(time.Minute * time.Duration(s.BillingIntervalQty))
	case "hour":
		rsp = base.Add(time.Hour * time.Duration(s.BillingIntervalQty))
	case "day":
		rsp = base.AddDate(0, 0, s.BillingIntervalQty)
	case "week":
		rsp = base.AddDate(0, 0, s.BillingIntervalQty*7)
	case "month":
		rsp = base.AddDate(0, s.BillingIntervalQty, 0)
	case "year":
		rsp = base.AddDate(s.BillingIntervalQty, 0, 0)
	}

	return rsp
}

func (s *Subscription) UpdateBillingAnchor(anchor int, prorationMode string) ProrationDetails {
	now := time.Now()
	year, month, _ := now.Date()

	// Use the current period start as the reference time to preserve time information
	referenceTime := s.CurrentPeriodStart

	// Try the current month first
	nextBilling := calculateBillingAnchor(anchor, year, int(month), referenceTime)

	// If the date has passed, move to next month
	if nextBilling.Before(now) {
		if month == 12 {
			year++
			month = 1
		} else {
			month++
		}
		nextBilling = calculateBillingAnchor(anchor, year, int(month), referenceTime)
	}

	// Store the old billing anchor for proration calculation
	oldBillingAnchor := s.BillingAnchor

	// Update the billing anchor
	s.BillingAnchor = anchor

	// Calculate the new billing period
	newPeriodStart := nextBilling
	newPeriodEnd := s.AddBillingInterval(newPeriodStart)

	// Calculate proration details
	details := s.CalculateProrationDetails(
		prorationMode,
		now,
		oldBillingAnchor,
		anchor,
		newPeriodStart,
		newPeriodEnd,
	)

	// Update the subscription's current period start and end
	s.CurrentPeriodStart = newPeriodStart
	s.CurrentPeriodEnd = newPeriodEnd
	// Update the renews at date
	s.RenewsAt = newPeriodEnd
	s.UpdatedAt = time.Now().UTC()
	return details
}

// SetActivation sets the activation date for a subscription based on the trial interval
func (s *Subscription) SetActivationDates() *Subscription {
	price := s.OrderItem.Price
	var startDate = time.Now().UTC()
	var trialEndsAt time.Time
	var endsAt time.Time

	if s.OrderItem.Price.TrialInterval != prices.BillingIntervalNone {
		switch s.OrderItem.Price.TrialInterval {
		case "minute":
			trialEndsAt = startDate.Add(time.Minute * time.Duration(s.OrderItem.Price.TrialIntervalQty))
		case "hour":
			trialEndsAt = startDate.Add(time.Hour * time.Duration(s.OrderItem.Price.TrialIntervalQty))
		case "day":
			trialEndsAt = startDate.AddDate(0, 0, s.OrderItem.Price.TrialIntervalQty)
		case "week":
			trialEndsAt = startDate.AddDate(0, 0, s.OrderItem.Price.TrialIntervalQty*7)
		case "month":
			trialEndsAt = startDate.AddDate(0, s.OrderItem.Price.TrialIntervalQty, 0)
		case "year":
			trialEndsAt = startDate.AddDate(s.OrderItem.Price.TrialIntervalQty, 0, 0)
		}
	}

	if s.OrderItem.Price.Cycles > 0 {
		endsAtV := calculateNextDate(price.BillingInterval, price.Cycles*price.BillingIntervalQty, startDate)
		endsAt = endsAtV
	}

	s.StartDate = startDate
	s.TrialEndsAt = trialEndsAt
	s.EndsAt = endsAt
	renewsAt := s.CalculateNextBillingDate()
	s.RenewsAt = renewsAt

	s.CurrentPeriodStart = startDate
	s.CurrentPeriodEnd = s.RenewsAt
	s.BillingAnchor = startDate.Day()

	return s
}

// SetActive sets the subscription status to active and prepares the dates for the subscription and charge schedule.
// It doesn't make any assumptions about the payment status, it just sets the subscription status to active. This
// calls SetActivationDates() and sets the status to SubscriptionStatusActive
func (s *Subscription) SetActive(payment Payment) *Subscription {
	s.SetActivationDates()
	s.Status = SubscriptionStatusActive
	if payment.OrgId != "" && payment.Amount > 0 {
		s.LastCharge = payment.CompletedAt
		s.TotalRevenue = payment.Amount
		s.CyclesProcessed++

		renewsAt := s.CalculateNextBillingDate()
		s.RenewsAt = renewsAt
		s.CurrentPeriodStart = s.StartDate
		s.CurrentPeriodEnd = renewsAt
	}

	return s
}

// SetCancelled sets the subscription status to cancelled and prepares the dates for the subscription and charge schedule.
func (s *Subscription) SetCancelled() *Subscription {
	s.Status = SubscriptionStatusCancelled
	s.CancelledAt = time.Now().UTC()
	s.RenewsAt = time.Time{}
	s.DunningActive = false
	s.ActiveDunningCampaignId = ""
	return s
}

// calculateBillingAnchor calculates the billing anchor date based on the anchor day, year, and month.
// if the anchor day is greater than the number of days in the month, it sets it to the last day of the month.
// It preserves the time information from the reference time.
func calculateBillingAnchor(anchor int, year int, month int, referenceTime time.Time) time.Time {
	daysInMonth := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()

	billingDay := anchor
	if anchor > daysInMonth {
		billingDay = daysInMonth
	}

	hour, min, sec := referenceTime.Clock()
	nsec := referenceTime.Nanosecond()

	return time.Date(year, time.Month(month), billingDay, hour, min, sec, nsec, time.UTC)
}

func calculateNextDate(interval prices.BillingInterval, qty int, startDate time.Time) time.Time {
	switch interval {
	case "minute":
		startDate = startDate.Add(time.Minute * time.Duration(qty))
	case "hour":
		startDate = startDate.Add(time.Hour * time.Duration(qty))
	case "day":
		startDate = startDate.AddDate(0, 0, qty)
	case "week":
		startDate = startDate.AddDate(0, 0, qty*7)
	case "month":
		startDate = startDate.AddDate(0, qty, 0)
	case "year":
		startDate = startDate.AddDate(qty, 0, 0)
	}
	return startDate
}

// NewSubscriptionFromItem creates a new Subscription from a payloop-cart Item
func NewSubscriptionFromOrderItem(item OrderItem) Subscription {
	subscription := Subscription{
		OrgId:              item.OrgId,
		Id:                 lib.GenerateId("sub"),
		OrderId:            item.OrderId,
		OrderItemId:        item.Id,
		OrderItem:          item,
		Status:             SubscriptionStatusPending,
		BillingInterval:    item.Price.BillingInterval,
		BillingIntervalQty: item.Price.BillingIntervalQty,
		Cycles:             item.Price.Cycles,
		DunningActive:      false,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	// Create a subscription item for backward compatibility
	subscriptionItem := NewSubscriptionItem(
		item.OrgId,
		subscription.Id,
		item.Price.Id,
		item.Description,
		string(item.Price.Currency),
	)
	subscriptionItem.ProductId = item.ProductId
	subscriptionItem.VariantId = item.VariantId
	subscriptionItem.Description = item.Description

	// Configure pricing based on category
	switch item.Price.Category {
	case prices.PriceCategoryUsage:
		// Pure usage-based billing
		subscriptionItem.HasUsage = true
		subscriptionItem.UsageType = UsageType(item.Price.UsageType)
		subscriptionItem.UnitType = UnitType(item.Price.UnitType)
		subscriptionItem.AggregationType = AggregationType(item.Price.AggregationType)
		subscriptionItem.UnitPrice = item.Price.UnitPrice
		subscriptionItem.PercentageRate = item.Price.PercentageRate
		subscriptionItem.FixedFee = item.Price.FixedFee
		subscriptionItem.Amount = 0 // No fixed amount for pure usage
	case prices.PriceCategoryHybrid:
		// Hybrid billing (fixed + usage)
		subscriptionItem.HasUsage = true
		subscriptionItem.UsageType = UsageType(item.Price.UsageType)
		subscriptionItem.UnitType = UnitType(item.Price.UnitType)
		subscriptionItem.AggregationType = AggregationType(item.Price.AggregationType)
		subscriptionItem.Amount = item.Price.UnitPrice * int64(item.Quantity) // Base fixed amount multiplied by quantity
		subscriptionItem.UnitPrice = item.Price.OverageUnitPrice // Overage rate
		subscriptionItem.PercentageRate = item.Price.PercentageRate
		subscriptionItem.FixedFee = item.Price.FixedFee
	default:
		// Traditional subscription billing
		subscriptionItem.Amount = item.Price.UnitPrice
		subscriptionItem.HasUsage = false
	}

	subscription.Items = []SubscriptionItem{subscriptionItem}

	return subscription
}

// NewSubscriptionFrominput creates a new Subscription from a payloop-cart input
func NewFromCreateInput(input CreateSubscriptionInput) Subscription {

	var startDate = time.Now().UTC()
	var trialEndsAt time.Time
	if input.TrialInterval != prices.BillingIntervalNone {
		switch input.TrialInterval {
		case "minute":
			startDate = startDate.Add(time.Minute * time.Duration(input.TrialIntervalQty))
		case "hour":
			startDate = startDate.Add(time.Hour * time.Duration(input.TrialIntervalQty))
		case "day":
			startDate = startDate.AddDate(0, 0, input.TrialIntervalQty)
		case "week":
			startDate = startDate.AddDate(0, 0, input.TrialIntervalQty*7)
		case "month":
			startDate = startDate.AddDate(0, input.TrialIntervalQty, 0)
		case "year":
			startDate = startDate.AddDate(input.TrialIntervalQty, 0, 0)
		}

		trialEndsAt = startDate
	}

	subscription := Subscription{
		OrgId:                   input.OrgId,
		Id:                      lib.GenerateId("sub"),
		Status:                  SubscriptionStatusPending,
		StartDate:               startDate,
		BillingInterval:         input.BillingInterval,
		BillingIntervalQty:      input.BillingIntervalQty,
		Cycles:                  0,
		BillingAnchor:           startDate.Day(),
		TrialEndsAt:             trialEndsAt,
		DunningActive:           false,
		ActiveDunningCampaignId: "",
		CyclesProcessed:         0,
		TotalRevenue:            0,
		CreatedAt:               time.Now().UTC(),
		UpdatedAt:               time.Now().UTC(),
	}

	// Create a subscription item for backward compatibility
	subscriptionItem := NewSubscriptionItem(
		input.OrgId,
		subscription.Id,
		"",             // No price ID available from input
		"Subscription", // Generic name
		input.Currency,
	)
	subscriptionItem.Amount = input.Amount

	subscription.Items = []SubscriptionItem{subscriptionItem}

	return subscription
}
