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

// Subscription is the recurring-billing aggregate root. Cross-aggregate
// references are by ID only — Customer / OrderItem / Price are separate
// aggregates and not embedded here. Use service.SubscriptionDetails when a
// query needs the composed view, or call the relevant repo's FindByIds.
type Subscription struct {
	OrgId           string
	Id              string
	PspId           Gateway
	OrderId         string
	CustomerId      string
	Status          SubscriptionStatus
	PaymentMethodId string

	StartDate          time.Time
	EndDate            time.Time
	BillingInterval    BillingInterval
	BillingIntervalQty int
	Cycles             int
	BillingAnchor      int

	// Trial cadence the subscription derived from its lines at construction, so it
	// can compute TrialEndsAt at activation without re-reading a Price.
	TrialInterval    BillingInterval
	TrialIntervalQty int

	TrialEndsAt time.Time
	CancelAt    time.Time
	EndsAt      time.Time
	LastCharge  time.Time
	RenewsAt    time.Time

	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time

	Retries     int
	NextRetryAt time.Time

	Currency        string
	Metadata        map[string]string
	CyclesProcessed int
	TotalRevenue    int64
	CancelledAt     time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

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

// CalculateProrationDetails computes credit details for a billing-anchor
// change. unitPriceMinor is the subscription's fixed-price slice in minor
// units (cents) — sourced by the caller from the linked Price.
func (s *Subscription) CalculateProrationDetails(
	prorationMode string,
	referenceDate time.Time,
	oldBillingAnchor, newBillingAnchor int,
	newPeriodStart, newPeriodEnd time.Time,
	unitPriceMinor int64,
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

		creditAmount := (unitPriceMinor * int64(daysRemaining)) / int64(totalDays)
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

// IsDueForBilling reports whether the subscription is due to be billed at the
// given instant. Engine-agnostic Go mirror of the SQL in
// SubscriptionRepository.FindDueForBilling — keep both in sync.
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

// UpdateBillingAnchor changes the subscription's billing anchor. unitPriceMinor
// is the linked Price's UnitPrice (in cents); the caller fetches it before
// invoking this method.
func (s *Subscription) UpdateBillingAnchor(anchor int, prorationMode string, unitPriceMinor int64) ProrationDetails {
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
		prorationMode, now, oldBillingAnchor, anchor, newPeriodStart, newPeriodEnd, unitPriceMinor,
	)

	s.CurrentPeriodStart = newPeriodStart
	s.CurrentPeriodEnd = newPeriodEnd
	s.RenewsAt = newPeriodEnd
	s.UpdatedAt = time.Now().UTC()
	return details
}

// SetActivationDates initializes the lifecycle date fields (StartDate /
// TrialEndsAt / EndsAt / RenewsAt / CurrentPeriodStart / CurrentPeriodEnd /
// BillingAnchor) from the subscription's OWN billing fields — cadence, cycles,
// and trial, all derived from its lines at construction (NewSubscriptionFromLines).
// No Price argument: the aggregate owns the truth.
func (s *Subscription) SetActivationDates() *Subscription {
	startDate := time.Now().UTC()
	var trialEndsAt time.Time
	var endsAt time.Time

	if s.TrialInterval != BillingIntervalNone && s.TrialInterval != "" {
		switch s.TrialInterval {
		case "minute":
			trialEndsAt = startDate.Add(time.Minute * time.Duration(s.TrialIntervalQty))
		case "hour":
			trialEndsAt = startDate.Add(time.Hour * time.Duration(s.TrialIntervalQty))
		case "day":
			trialEndsAt = startDate.AddDate(0, 0, s.TrialIntervalQty)
		case "week":
			trialEndsAt = startDate.AddDate(0, 0, s.TrialIntervalQty*7)
		case "month":
			trialEndsAt = startDate.AddDate(0, s.TrialIntervalQty, 0)
		case "year":
			trialEndsAt = startDate.AddDate(s.TrialIntervalQty, 0, 0)
		}
	}

	if s.Cycles > 0 {
		endsAt = calculateNextDate(s.BillingInterval, s.Cycles*s.BillingIntervalQty, startDate)
	}

	s.StartDate = startDate
	s.TrialEndsAt = trialEndsAt
	s.EndsAt = endsAt
	s.RenewsAt = s.CalculateNextBillingDate()
	s.CurrentPeriodStart = startDate
	s.CurrentPeriodEnd = s.RenewsAt
	s.BillingAnchor = startDate.Day()

	return s
}

// SetActive transitions the subscription to active, initializing its lifecycle
// dates from its own fields. payment is the (possibly zero-value) first-cycle
// Payment reflected in LastCharge / TotalRevenue / CyclesProcessed.
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

// NewSubscriptionFromLines constructs a pending Subscription from a group of the
// order's recurring lines that share one billing cadence (linked later via
// OrderItem.SubscriptionId). The subscription derives everything it needs from
// its own lines — cadence, cycles cap, and trial — rather than having an outside
// caller pick a representative price. The plan line (the first non-metered line,
// else the first line) supplies cycles + trial; the cadence is the lines' shared
// SubscriptionCadence (metered lines capped at monthly). It stores no charge
// amount (ADR 0002) — the per-cycle total is computed onto the Invoice.
func NewSubscriptionFromLines(orgId, orderId, customerId string, prices []Price) Subscription {
	plan := prices[0]
	for _, p := range prices {
		if !p.IsMetered() {
			plan = p
			break
		}
	}
	interval, qty := plan.SubscriptionCadence()
	return Subscription{
		OrgId:              orgId,
		Id:                 lib.GenerateId("sub"),
		OrderId:            orderId,
		CustomerId:         customerId,
		Status:             SubscriptionStatusPending,
		BillingInterval:    interval,
		BillingIntervalQty: qty,
		Cycles:             plan.Cycles,
		TrialInterval:      plan.TrialInterval,
		TrialIntervalQty:   plan.TrialIntervalQty,
		Currency:           string(plan.Currency),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}
