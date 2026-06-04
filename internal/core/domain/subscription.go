package domain

import (
	"getpaidhq/internal/lib"
	"maps"
	"time"
)

type SubscriptionStatus string

const (
	SubscriptionStatusTrial       SubscriptionStatus = "trial"
	SubscriptionStatusActive      SubscriptionStatus = "active"
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
	OrgId           string             `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id              string             `gorm:"column:id;primaryKey" json:"id"`
	PspId           Gateway            `gorm:"column:psp_id" json:"psp_id"`
	OrderId         string             `gorm:"column:order_id" json:"order_id"`
	OrderItemId     string             `gorm:"column:order_item_id" json:"order_item_id"`
	OrderItem       OrderItem          `gorm:"foreignKey:OrderItemId,OrgId;references:Id,OrgId" json:"-"`
	CustomerId      string             `gorm:"column:customer_id" json:"customer_id"`
	Customer        Customer           `gorm:"foreignKey:CustomerId,OrgId;references:Id,OrgId" json:"-"`
	Status          SubscriptionStatus `gorm:"column:status" json:"status"`
	PaymentMethodId string             `gorm:"column:payment_method_id" json:"payment_method_id,omitempty"`

	StartDate          time.Time       `gorm:"column:start_date;serializer:nulltime" json:"start_date"`
	EndDate            time.Time       `gorm:"column:end_date;serializer:nulltime" json:"end_date,omitzero"`
	BillingInterval    BillingInterval `gorm:"column:billing_interval" json:"billing_interval"`
	BillingIntervalQty int             `gorm:"column:billing_interval_qty" json:"billing_interval_qty"`
	Cycles             int             `gorm:"column:cycles" json:"cycles"`
	BillingAnchor      int             `gorm:"column:billing_anchor" json:"billing_anchor"`

	TrialEndsAt time.Time `gorm:"column:trial_ends_at;serializer:nulltime" json:"trial_ends_at,omitzero"`
	CancelAt    time.Time `gorm:"column:cancel_at;serializer:nulltime" json:"cancel_at,omitzero"`
	EndsAt      time.Time `gorm:"column:ends_at;serializer:nulltime" json:"ends_at,omitzero"`
	LastCharge  time.Time `gorm:"column:last_charge;serializer:nulltime" json:"last_charge"`
	RenewsAt    time.Time `gorm:"column:renews_at;serializer:nulltime" json:"renews_at"`

	CurrentPeriodStart time.Time `gorm:"column:current_period_start;serializer:nulltime" json:"current_period_start"`
	CurrentPeriodEnd   time.Time `gorm:"column:current_period_end;serializer:nulltime" json:"current_period_end"`

	Retries     int       `gorm:"column:retries" json:"retries"`
	NextRetryAt time.Time `gorm:"column:next_retry;serializer:nulltime" json:"next_retry,omitzero"`

	Currency        string            `gorm:"column:currency" json:"currency"`
	Amount          int64             `gorm:"column:amount" json:"amount"`
	Metadata        map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CyclesProcessed int               `gorm:"column:cycles_processed" json:"cycles_processed"`
	TotalRevenue    int64             `gorm:"column:total_revenue" json:"total_revenue"`
	CancelledAt     time.Time         `gorm:"column:cancelled_at;serializer:nulltime" json:"cancelled_at,omitzero"`
	CreatedAt       time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Subscription) TableName() string { return "subscriptions" }

type ProrationDetails struct {
	CreditAmount       int       `json:"credit_amount"`
	DaysCredited       int       `json:"days_credited"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	OldBillingAnchor   int       `json:"old_billing_anchor,omitempty"`
	NewBillingAnchor   int       `json:"new_billing_anchor,omitempty"`
	NewPeriodStart     time.Time `json:"new_period_start"`
	NewPeriodEnd       time.Time `json:"new_period_end"`
}

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

	if prorationMode == "none" {
		return details
	}

	if prorationMode == "credit_unused" {
		totalDays := int(s.CurrentPeriodEnd.Sub(s.CurrentPeriodStart).Hours() / 24)
		if totalDays <= 0 {
			return details
		}

		daysRemaining := int(s.CurrentPeriodEnd.Sub(referenceDate).Hours() / 24)
		if daysRemaining <= 0 {
			return details
		}

		// Stay in integer minor-units — a float round-trip would lose cents
		// asymmetrically and produce off-by-one discrepancies at the cent
		// boundary. Same truncation semantics as before (toward zero), which
		// favors the merchant slightly; if business wants banker's rounding
		// or ceiling, change here in one place.
		creditAmount := (s.Amount * int64(daysRemaining)) / int64(totalDays)
		details.CreditAmount = int(creditAmount)
		details.DaysCredited = daysRemaining
	}

	return details
}

func (s *Subscription) IsRunning() bool {
	return s.Status == SubscriptionStatusActive ||
		s.Status == SubscriptionStatusTrial ||
		s.Status == SubscriptionStatusPastDue
}

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

// IsDueForBilling reports whether the subscription is due to be billed at the given
// instant. It is the engine-agnostic Go mirror of the SQL in
// SubscriptionRepository.FindDueForBilling (postgres/subscription_repo.go) and the two
// MUST stay in sync — the SQL is the hourly sweep's selection rule, this is what the
// Hatchet activation-spawn uses to decide whether to kick off an immediate first charge.
//
// Due when any of:
//   - active with a non-zero RenewsAt that is now-or-past, or
//   - past_due with a non-zero NextRetryAt that is now-or-past, or
//   - trial with a non-zero TrialEndsAt that is now-or-past.
//
// Zero (unset) dates map to NULL in the SQL and `col <= now` is false for NULL, so
// they are never due — the `!X.IsZero()` guards mirror that exclusion.
func (s *Subscription) IsDueForBilling(now time.Time) bool {
	switch s.Status {
	case SubscriptionStatusActive:
		return !s.RenewsAt.IsZero() && !s.RenewsAt.After(now)
	case SubscriptionStatusPastDue:
		return !s.NextRetryAt.IsZero() && !s.NextRetryAt.After(now)
	case SubscriptionStatusTrial:
		return !s.TrialEndsAt.IsZero() && !s.TrialEndsAt.After(now)
	default:
		return false
	}
}

func (s *Subscription) SetMetadata(meta map[string]string) *Subscription {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	maps.Copy(s.Metadata, meta)
	return s
}

func (s *Subscription) CalculateNextBillingDate() time.Time {
	if s.BillingInterval == "" || s.BillingIntervalQty <= 0 {
		return time.Time{}
	}
	var nextBillingDate time.Time
	if s.LastCharge.IsZero() && s.CyclesProcessed == 0 {
		return s.StartDate
	}

	base := s.CurrentPeriodEnd
	if base.IsZero() {
		// Recurring charge before a period boundary has been established
		// (e.g. CurrentPeriodEnd not yet set/persisted): advance from
		// StartDate instead of the zero time, which would otherwise produce
		// a year-0001 billing date.
		base = s.StartDate
	}

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
	referenceTime := s.CurrentPeriodStart

	nextBilling := calculateBillingAnchor(anchor, year, int(month), referenceTime)
	if nextBilling.Before(now) {
		if month == 12 {
			year++
			month = 1
		} else {
			month++
		}
		nextBilling = calculateBillingAnchor(anchor, year, int(month), referenceTime)
	}

	oldBillingAnchor := s.BillingAnchor
	s.BillingAnchor = anchor

	newPeriodStart := nextBilling
	newPeriodEnd := s.AddBillingInterval(newPeriodStart)

	details := s.CalculateProrationDetails(
		prorationMode, now, oldBillingAnchor, anchor, newPeriodStart, newPeriodEnd,
	)

	s.CurrentPeriodStart = newPeriodStart
	s.CurrentPeriodEnd = newPeriodEnd
	s.RenewsAt = newPeriodEnd
	s.UpdatedAt = time.Now().UTC()
	return details
}

func (s *Subscription) SetActivationDates() *Subscription {
	price := s.OrderItem.Price
	var startDate = time.Now().UTC()
	var trialEndsAt time.Time
	var endsAt time.Time

	if s.OrderItem.Price.TrialInterval != BillingIntervalNone {
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

func (s *Subscription) SetCancelled() *Subscription {
	s.Status = SubscriptionStatusCancelled
	s.CancelledAt = time.Now().UTC()
	s.RenewsAt = time.Time{}
	s.NextRetryAt = time.Time{}
	return s
}

func calculateBillingAnchor(anchor int, year int, month int, referenceTime time.Time) time.Time {
	daysInMonth := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
	billingDay := min(anchor, daysInMonth)
	hour, min, sec := referenceTime.Clock()
	nsec := referenceTime.Nanosecond()
	return time.Date(year, time.Month(month), billingDay, hour, min, sec, nsec, time.UTC)
}

func calculateNextDate(interval BillingInterval, qty int, startDate time.Time) time.Time {
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
