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

	// Retries is the number of times the subscription has been retried for payment in the current billing cycle.
	Retries int `json:"retries"`
	// NextRetryAt is the date when the subscription will be retried for payment next.
	// Only used if the subscription is in PastDue state.
	NextRetryAt time.Time `json:"next_retry,omitempty,omitzero"`

	Currency        string            `json:"currency"`
	Amount          int64             `json:"amount"`
	Metadata        map[string]string `json:"metadata"`
	CyclesProcessed int               `json:"cycles_processed"`
	TotalRevenue    int64             `json:"total_revenue"`
	CancelledAt     time.Time         `json:"cancelled_at,omitempty,omitzero"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// IsRunning checks if the subscription is in an active or trial state.
func (s *Subscription) IsRunning() bool {
	return s.Status == SubscriptionStatusActive ||
		s.Status == SubscriptionStatusTrial ||
		s.Status == SubscriptionStatusPastDue
}

// GetNextChargeDate returns the next charge date for the subscription, which is the earlier of RenewsAt or NextRetryAt,
// and only if the subscription is in Active or PastDue state.
func (s *Subscription) GetNextChargeDate() time.Time {
	if s.Status == SubscriptionStatusPastDue {
		return s.NextRetryAt
	}

	if s.Status == SubscriptionStatusActive {
		return s.RenewsAt
	}

	if s.RenewsAt.Before(s.NextRetryAt) {
		return s.RenewsAt
	}

	return s.NextRetryAt
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
	s.NextRetryAt = time.Time{}
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
		Retries:            0,
		Currency:           string(item.Price.Currency),
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
		TrialEndsAt:        trialEndsAt,
		Retries:            0,
		Currency:           input.Currency,
		Amount:             input.Amount,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}
