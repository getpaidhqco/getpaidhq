package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// subscriptionRow is the postgres on-the-wire shape of a Subscription. Customer
// and OrderItem are NOT embedded — composition is a service-layer concern; see
// service.SubscriptionDetails and the Customer / OrderItem repos.
type subscriptionRow struct {
	OrgId           string                    `gorm:"column:org_id;primaryKey"`
	Id              string                    `gorm:"column:id;primaryKey"`
	PspId           domain.Gateway            `gorm:"column:psp_id"`
	OrderId         string                    `gorm:"column:order_id"`
	CustomerId      string                    `gorm:"column:customer_id"`
	Status          domain.SubscriptionStatus `gorm:"column:status"`
	PaymentMethodId string                    `gorm:"column:payment_method_id"`

	StartDate          time.Time              `gorm:"column:start_date;serializer:nulltime"`
	EndDate            time.Time              `gorm:"column:end_date;serializer:nulltime"`
	BillingInterval    domain.BillingInterval `gorm:"column:billing_interval"`
	BillingIntervalQty int                    `gorm:"column:billing_interval_qty"`
	Cycles             int                    `gorm:"column:cycles"`
	BillingAnchor      int                    `gorm:"column:billing_anchor"`
	TrialInterval      domain.BillingInterval `gorm:"column:trial_interval"`
	TrialIntervalQty   int                    `gorm:"column:trial_interval_qty"`

	TrialEndsAt time.Time `gorm:"column:trial_ends_at;serializer:nulltime"`
	CancelAt    time.Time `gorm:"column:cancel_at;serializer:nulltime"`
	EndsAt      time.Time `gorm:"column:ends_at;serializer:nulltime"`
	LastCharge  time.Time `gorm:"column:last_charge;serializer:nulltime"`
	RenewsAt    time.Time `gorm:"column:renews_at;serializer:nulltime"`

	CurrentPeriodStart time.Time `gorm:"column:current_period_start;serializer:nulltime"`
	CurrentPeriodEnd   time.Time `gorm:"column:current_period_end;serializer:nulltime"`

	Retries     int       `gorm:"column:retries"`
	NextRetryAt time.Time `gorm:"column:next_retry;serializer:nulltime"`

	Currency        string            `gorm:"column:currency"`
	Metadata        map[string]string `gorm:"column:metadata;serializer:json"`
	CyclesProcessed int               `gorm:"column:cycles_processed"`
	TotalRevenue    int64             `gorm:"column:total_revenue"`
	CancelledAt     time.Time         `gorm:"column:cancelled_at;serializer:nulltime"`
	CreatedAt       time.Time         `gorm:"column:created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at"`
}

func (subscriptionRow) TableName() string { return "subscriptions" }

func (r subscriptionRow) toDomain() domain.Subscription {
	return domain.Subscription{
		OrgId:              r.OrgId,
		Id:                 r.Id,
		PspId:              r.PspId,
		OrderId:            r.OrderId,
		CustomerId:         r.CustomerId,
		Status:             r.Status,
		PaymentMethodId:    r.PaymentMethodId,
		StartDate:          r.StartDate,
		EndDate:            r.EndDate,
		BillingInterval:    r.BillingInterval,
		BillingIntervalQty: r.BillingIntervalQty,
		Cycles:             r.Cycles,
		BillingAnchor:      r.BillingAnchor,
		TrialInterval:      r.TrialInterval,
		TrialIntervalQty:   r.TrialIntervalQty,
		TrialEndsAt:        r.TrialEndsAt,
		CancelAt:           r.CancelAt,
		EndsAt:             r.EndsAt,
		LastCharge:         r.LastCharge,
		RenewsAt:           r.RenewsAt,
		CurrentPeriodStart: r.CurrentPeriodStart,
		CurrentPeriodEnd:   r.CurrentPeriodEnd,
		Retries:            r.Retries,
		NextRetryAt:        r.NextRetryAt,
		Currency:           r.Currency,
		Metadata:           r.Metadata,
		CyclesProcessed:    r.CyclesProcessed,
		TotalRevenue:       r.TotalRevenue,
		CancelledAt:        r.CancelledAt,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

func subscriptionRowFromDomain(s domain.Subscription) subscriptionRow {
	return subscriptionRow{
		OrgId:              s.OrgId,
		Id:                 s.Id,
		PspId:              s.PspId,
		OrderId:            s.OrderId,
		CustomerId:         s.CustomerId,
		Status:             s.Status,
		PaymentMethodId:    s.PaymentMethodId,
		StartDate:          s.StartDate,
		EndDate:            s.EndDate,
		BillingInterval:    s.BillingInterval,
		BillingIntervalQty: s.BillingIntervalQty,
		Cycles:             s.Cycles,
		BillingAnchor:      s.BillingAnchor,
		TrialInterval:      s.TrialInterval,
		TrialIntervalQty:   s.TrialIntervalQty,
		TrialEndsAt:        s.TrialEndsAt,
		CancelAt:           s.CancelAt,
		EndsAt:             s.EndsAt,
		LastCharge:         s.LastCharge,
		RenewsAt:           s.RenewsAt,
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		Retries:            s.Retries,
		NextRetryAt:        s.NextRetryAt,
		Currency:           s.Currency,
		Metadata:           s.Metadata,
		CyclesProcessed:    s.CyclesProcessed,
		TotalRevenue:       s.TotalRevenue,
		CancelledAt:        s.CancelledAt,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
}

func subscriptionRowsToDomain(rows []subscriptionRow) []domain.Subscription {
	out := make([]domain.Subscription, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
