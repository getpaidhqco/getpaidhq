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

	// Product, variant and price references
	ProductId string `json:"product_id"`
	VariantId string `json:"variant_id"`
	PriceId   string `json:"price_id"`

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

	Currency        string            `json:"currency"`
	Amount          int64             `json:"amount"`
	Metadata        map[string]string `json:"metadata"`
	CyclesProcessed int               `json:"cycles_processed"`
	TotalRevenue    int64             `json:"total_revenue"`
	CancelledAt     time.Time         `json:"cancelled_at,omitempty,omitzero"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
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

		// Calculate the credit amount based on the proportion of days remaining
		creditAmount := int(float64(s.Amount) * float64(daysRemaining) / float64(totalDays))

		details.CreditAmount = creditAmount
		details.DaysCredited = daysRemaining
	}

	return details
}

// IsRunning checks if the subscription is in an active or trial state.
func (s *Subscription) IsRunning() bool {
	return s.Status == SubscriptionStatusActive ||
		s.Status == SubscriptionStatusTrial ||
		s.Status == SubscriptionStatusPastDue
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

	return Subscription{
		OrgId:              item.OrgId,
		Id:                 lib.GenerateId("sub"),
		OrderId:            item.OrderId,
		OrderItemId:        item.Id,
		OrderItem:          item,
		ProductId:          item.ProductId,
		VariantId:          item.VariantId,
		PriceId:            item.Price.Id,
		Status:             SubscriptionStatusPending,
		BillingInterval:    item.Price.BillingInterval,
		BillingIntervalQty: item.Price.BillingIntervalQty,
		Cycles:                 item.Price.Cycles,
		DunningActive:         false,
		ActiveDunningCampaignId: "",
		Currency:               string(item.Price.Currency),
		Amount:             item.Price.UnitPrice,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
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

	return Subscription{
		OrgId:              input.OrgId,
		Id:                 lib.GenerateId("sub"),
		Status:             SubscriptionStatusPending,
		StartDate:          startDate,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		Cycles:             0,
		BillingAnchor:      startDate.Day(),
		TrialEndsAt:            trialEndsAt,
		DunningActive:          false,
		ActiveDunningCampaignId: "",
		Currency:               input.Currency,
		Amount:             input.Amount,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}
