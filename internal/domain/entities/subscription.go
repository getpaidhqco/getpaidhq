package entities

import (
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/lib"
	"time"
)

type CreateSubscriptionInput struct {
	OrgId string `json:"org_id"`

	PaymentMethodId string `json:"payment_method_id" binding:"required"`
	Activate        bool   `json:"activate"`

	Amount   int    `json:"amount"  binding:"required"`
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

	// The initial schedule charge failed, so the subscription is in a retry workflow
	// that will attempt to charge the customer again.  The retry workflow can't be
	// longer than the subscription period, so if the retry fails, the subscription
	// will be marked as past_due
	SubscriptionStatusRetry SubscriptionStatus = "retry"

	// Payment failed, and not being retried, so waiting to be renewed or cancelled
	SubscriptionStatusPastDue     SubscriptionStatus = "past_due"
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
	OrgId              string                 `json:"org_id"`
	Id                 string                 `json:"id"`
	OrderId            string                 `json:"order_id"`
	OrderItemId        string                 `json:"order_item_id"`
	OrderItem          OrderItem              `json:"-"`
	CustomerId         string                 `json:"customer_id"`
	Status             SubscriptionStatus     `json:"status"`
	PaymentMethodId    *string                `json:"payment_method_id,omitempty"`
	StartDate          time.Time              `json:"start_date"`
	EndDate            *time.Time             `json:"end_date"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	Cycles             int                    `json:"cycles"`
	BillingAnchor      int                    `json:"billing_anchor"`
	TrialEndsAt        *time.Time             `json:"trial_ends_at"`
	CancelAt           *time.Time             `json:"cancel_at"`
	EndsAt             *time.Time             `json:"ends_at"`
	LastCharge         *time.Time             `json:"last_charge"`
	RenewsAt           *time.Time             `json:"renews_at"`

	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`

	Retries     int        `json:"retries"`
	NextRetryAt *time.Time `json:"next_retry"`

	Currency        string            `json:"currency"`
	Amount          int               `json:"amount"`
	Metadata        map[string]string `json:"metadata"`
	CyclesProcessed int               `json:"cycles_processed"`
	TotalRevenue    int               `json:"total_revenue"`
	CancelledAt     *time.Time        `json:"cancelled_at"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// CalculateNextBillingDate calculates and returns the next billing date for a subscription
// based on the StartDate, LastCharge, BillingInterval, BillingIntervalQty and CyclesProcessed
// If the subscription has not started yet, it returns the StartDate
// If the subscription has started but has not been charged yet, it returns the StartDate
// If the subscription has been charged, it uses the LastCharge date as the base date
// and the BillingInterval and BillingIntervalQty
//
// If the subscription is in retry status, it calculates the next retry date
func (s Subscription) CalculateNextBillingDate() time.Time {
	if s.BillingInterval == "" || s.BillingIntervalQty <= 0 {
		return time.Time{}
	}

	var nextBillingDate time.Time
	if s.Status == SubscriptionStatusRetry {
		// Next retry date is in the future
		if s.NextRetryAt != nil && s.NextRetryAt.After(time.Now().UTC()) {
			return *s.NextRetryAt
		}

		// Next retry already happened, use as base
		if s.NextRetryAt != nil && s.NextRetryAt.Before(time.Now().UTC()) {
			nextBillingDate = time.Now().UTC()
		} else {
			// Retry hasn't happened yet, use last charge date as base
			nextBillingDate = *s.LastCharge
		}

		return nextBillingDate.Add(time.Minute * 1)
	}

	nextBillingDate = s.StartDate
	if s.LastCharge == nil && s.CyclesProcessed == 0 {
		return nextBillingDate
	}

	if s.LastCharge != nil && s.LastCharge.After(nextBillingDate) {
		nextBillingDate = *s.LastCharge
	}

	switch s.BillingInterval {
	case "minute":
		nextBillingDate = nextBillingDate.Add(time.Minute * time.Duration(s.BillingIntervalQty))
	case "hour":
		nextBillingDate = nextBillingDate.Add(time.Hour * time.Duration(s.BillingIntervalQty))
	case "day":
		nextBillingDate = nextBillingDate.AddDate(0, 0, s.BillingIntervalQty)
	case "week":
		nextBillingDate = nextBillingDate.AddDate(0, 0, s.BillingIntervalQty*7)
	case "month":
		nextBillingDate = nextBillingDate.AddDate(0, s.BillingIntervalQty, 0)
	case "year":
		nextBillingDate = nextBillingDate.AddDate(s.BillingIntervalQty, 0, 0)
	}

	return nextBillingDate
}

// SetActivation sets the activation date for a subscription based on the trial interval
func (s *Subscription) SetActivationDates() *Subscription {
	price := s.OrderItem.Price
	var startDate = time.Now().UTC()
	var trialEndsAt *time.Time
	var endsAt *time.Time
	if s.OrderItem.Price.TrialInterval != prices.BillingIntervalNone {
		switch s.OrderItem.Price.TrialInterval {
		case "minute":
			startDate = startDate.Add(time.Minute * time.Duration(s.OrderItem.Price.TrialIntervalQty))
		case "hour":
			startDate = startDate.Add(time.Hour * time.Duration(s.OrderItem.Price.TrialIntervalQty))
		case "day":
			startDate = startDate.AddDate(0, 0, s.OrderItem.Price.TrialIntervalQty)
		case "week":
			startDate = startDate.AddDate(0, 0, s.OrderItem.Price.TrialIntervalQty*7)
		case "month":
			startDate = startDate.AddDate(0, s.OrderItem.Price.TrialIntervalQty, 0)
		case "year":
			startDate = startDate.AddDate(s.OrderItem.Price.TrialIntervalQty, 0, 0)
		}

		trialEndsAt = &startDate
	}

	if s.OrderItem.Price.Cycles > 0 {
		endsAtV := calculateNextDate(price.BillingInterval, price.Cycles*price.BillingIntervalQty, startDate)
		endsAt = &endsAtV
	}

	s.TrialEndsAt = trialEndsAt
	s.EndsAt = endsAt
	s.RenewsAt = &startDate
	s.StartDate = startDate

	return s
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
		Status:             SubscriptionStatusPending,
		BillingInterval:    item.Price.BillingInterval,
		BillingIntervalQty: item.Price.BillingIntervalQty,
		Cycles:             item.Price.Cycles,
		CancelAt:           nil,
		LastCharge:         nil,
		Retries:            0,
		NextRetryAt:        nil,
		Currency:           item.Price.Currency,
		Amount:             item.Price.UnitPrice,
		Metadata:           nil,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CancelledAt:        nil,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

// NewSubscriptionFrominput creates a new Subscription from a payloop-cart input
func NewFromCreateInput(input CreateSubscriptionInput) Subscription {

	var startDate = time.Now().UTC()
	var trialEndsAt *time.Time
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

		trialEndsAt = &startDate
	}

	return Subscription{
		OrgId:              input.OrgId,
		Id:                 lib.GenerateId("sub"),
		Status:             SubscriptionStatusPending,
		StartDate:          startDate,
		EndDate:            nil,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		Cycles:             0,
		BillingAnchor:      startDate.Day(),
		TrialEndsAt:        trialEndsAt,
		CancelAt:           nil,
		EndsAt:             nil,
		LastCharge:         nil,
		RenewsAt:           nil,
		Retries:            0,
		NextRetryAt:        nil,
		Currency:           input.Currency,
		Amount:             input.Amount,
		Metadata:           nil,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CancelledAt:        nil,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

func GetTopicFromStatus(status SubscriptionStatus) string {
	switch status {
	case SubscriptionStatusActive:
		return topic.TopicSubscriptionActivated
	case SubscriptionStatusPaused:
		return topic.TopicSubscriptionPaused
	case SubscriptionStatusCancelled:
		return topic.TopicSubscriptionCancelled
	case SubscriptionStatusExpired:
		return topic.SubscriptionStatusExpired
	case SubscriptionStatusPastDue:
		return topic.SubscriptionStatusExpired

	default:
		return ""
	}
}
