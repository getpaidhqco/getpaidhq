package domain

import (
	"errors"
	"payloop/internal/lib"
	"time"
)

const hoursPerDay = 24

type CreateSubscriptionInput struct {
	OrgId string `json:"org_id"`

	PaymentMethodId string `json:"payment_method_id" binding:"required"`
	Activate        bool   `json:"activate"`

	Amount   int64    `json:"amount" binding:"required"`
	Currency Currency `json:"currency" binding:"required"`

	BillingInterval    BillingInterval `json:"billing_interval" binding:"required"`
	BillingIntervalQty int             `json:"billing_interval_qty" binding:"required"`
	Cycles             int             `json:"cycles"`

	TrialInterval    BillingInterval `json:"trial_interval"`
	TrialIntervalQty int             `json:"trial_interval_qty"`

	Metadata map[string]string `json:"metadata"`
}

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

	StartDate          time.Time       `gorm:"column:start_date" json:"start_date"`
	EndDate            time.Time       `gorm:"column:end_date" json:"end_date,omitempty,omitzero"`
	BillingInterval    BillingInterval `gorm:"column:billing_interval" json:"billing_interval"`
	BillingIntervalQty int             `gorm:"column:billing_interval_qty" json:"billing_interval_qty"`
	Cycles             int             `gorm:"column:cycles" json:"cycles"`
	BillingAnchor      int             `gorm:"column:billing_anchor" json:"billing_anchor"`

	TrialEndsAt time.Time `gorm:"column:trial_ends_at" json:"trial_ends_at,omitempty,omitzero"`
	CancelAt    time.Time `gorm:"column:cancel_at" json:"cancel_at,omitempty,omitzero"`
	EndsAt      time.Time `gorm:"column:ends_at" json:"ends_at,omitempty,omitzero"`
	LastCharge  time.Time `gorm:"column:last_charge" json:"last_charge"`
	RenewsAt    time.Time `gorm:"column:renews_at" json:"renews_at"`

	CurrentPeriodStart time.Time `gorm:"column:current_period_start" json:"current_period_start"`
	CurrentPeriodEnd   time.Time `gorm:"column:current_period_end" json:"current_period_end"`

	Retries     int       `gorm:"column:retries" json:"retries"`
	NextRetryAt time.Time `gorm:"column:next_retry" json:"next_retry,omitempty,omitzero"`

	Currency        Currency          `gorm:"column:currency" json:"currency"`
	Amount          int64             `gorm:"column:amount" json:"amount"`
	Metadata        map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CyclesProcessed int               `gorm:"column:cycles_processed" json:"cycles_processed"`
	TotalRevenue    int64             `gorm:"column:total_revenue" json:"total_revenue"`
	CancelledAt     time.Time         `gorm:"column:cancelled_at" json:"cancelled_at,omitempty,omitzero"`
	CreatedAt       time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Subscription) TableName() string { return "subscriptions" }

func (s *Subscription) Validate() error {
	if s.OrgId == "" {
		return errors.New("org_id is required")
	}
	if s.Id == "" {
		return errors.New("id is required")
	}
	if s.Amount < 0 {
		return errors.New("amount must not be negative")
	}
	if s.BillingInterval == "" {
		return errors.New("billing_interval is required")
	}
	if s.BillingIntervalQty <= 0 {
		return errors.New("billing_interval_qty must be positive")
	}
	return nil
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
		totalDays := int(s.CurrentPeriodEnd.Sub(s.CurrentPeriodStart).Hours() / hoursPerDay)
		if totalDays <= 0 {
			return details
		}

		daysRemaining := int(s.CurrentPeriodEnd.Sub(referenceDate).Hours() / hoursPerDay)
		if daysRemaining <= 0 {
			return details
		}

		creditAmount := int(float64(s.Amount) * float64(daysRemaining) / float64(totalDays))
		details.CreditAmount = creditAmount
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

func (s *Subscription) SetMetadata(meta map[string]string) *Subscription {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	for key, value := range meta {
		s.Metadata[key] = value
	}
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
	billingDay := anchor
	if anchor > daysInMonth {
		billingDay = daysInMonth
	}
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
		Currency:           item.Price.Currency,
		Amount:             item.Price.UnitPrice,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

func NewFromCreateInput(input CreateSubscriptionInput) Subscription {
	var startDate = time.Now().UTC()
	var trialEndsAt time.Time
	if input.TrialInterval != BillingIntervalNone {
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
