package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// subscriptionRow is the postgres on-the-wire shape of a Subscription. Customer
// and OrderItem are NOT embedded — composition is a service-layer concern; see
// service.SubscriptionDetails and the Customer / OrderItem repos.
//
// The nullable timestamp columns (start_date, end_date, trial_ends_at, cancel_at,
// ends_at, last_charge, renews_at, current_period_start, current_period_end,
// next_retry, cancelled_at) are held as *time.Time: a zero domain time maps to
// SQL NULL on write (nullTime) and a NULL column maps back to the zero time on
// read (timeOrZero).
// payment_method_id is a nullable FK held as *string. retries is a nullable
// INTEGER held as *int. These two map through nilIfEmpty/strOrEmpty and a
// nil→0 conversion respectively at the domain boundary.
type subscriptionRow struct {
	OrgId           string
	Id              string
	PspId           string
	OrderId         string
	CustomerId      string
	Status          string
	PaymentMethodId *string

	StartDate          *time.Time
	EndDate            *time.Time
	BillingInterval    string
	BillingIntervalQty int
	Cycles             int
	BillingAnchor      int
	TrialInterval      string
	TrialIntervalQty   int

	TrialEndsAt *time.Time
	CancelAt    *time.Time
	EndsAt      *time.Time
	LastCharge  *time.Time
	RenewsAt    *time.Time

	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time

	Retries     *int
	NextRetryAt *time.Time

	Currency        string
	Metadata        jsonCol[map[string]string]
	CyclesProcessed int
	TotalRevenue    int64
	CancelledAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// subscriptionColumns lists the subscriptions table columns in the order the row
// scans and the INSERT binds them. next_retry is the physical column for
// NextRetryAt.
const subscriptionColumns = `org_id, id, psp_id, order_id, customer_id, status, payment_method_id,
	start_date, end_date, billing_interval, billing_interval_qty, cycles, billing_anchor,
	trial_interval, trial_interval_qty, trial_ends_at, cancel_at, ends_at, last_charge, renews_at,
	current_period_start, current_period_end, retries, next_retry, currency, metadata,
	cycles_processed, total_revenue, cancelled_at, created_at, updated_at`

// subscriptionSelectQualified is subscriptionColumns with every column
// prefixed by the table name, for SELECTs that JOIN other tables (where bare
// org_id/id/etc. would be ambiguous). Column order matches subscriptionColumns
// so the same scanInto applies.
const subscriptionSelectQualified = `subscriptions.org_id, subscriptions.id, subscriptions.psp_id,
	subscriptions.order_id, subscriptions.customer_id, subscriptions.status, subscriptions.payment_method_id,
	subscriptions.start_date, subscriptions.end_date, subscriptions.billing_interval, subscriptions.billing_interval_qty,
	subscriptions.cycles, subscriptions.billing_anchor, subscriptions.trial_interval, subscriptions.trial_interval_qty,
	subscriptions.trial_ends_at, subscriptions.cancel_at, subscriptions.ends_at, subscriptions.last_charge,
	subscriptions.renews_at, subscriptions.current_period_start, subscriptions.current_period_end, subscriptions.retries,
	subscriptions.next_retry, subscriptions.currency, subscriptions.metadata, subscriptions.cycles_processed,
	subscriptions.total_revenue, subscriptions.cancelled_at, subscriptions.created_at, subscriptions.updated_at`

func (r *subscriptionRow) scanInto(s scanner) error {
	return s.Scan(
		&r.OrgId, &r.Id, &r.PspId, &r.OrderId, &r.CustomerId, &r.Status, &r.PaymentMethodId,
		&r.StartDate, &r.EndDate, &r.BillingInterval, &r.BillingIntervalQty, &r.Cycles, &r.BillingAnchor,
		&r.TrialInterval, &r.TrialIntervalQty, &r.TrialEndsAt, &r.CancelAt, &r.EndsAt, &r.LastCharge, &r.RenewsAt,
		&r.CurrentPeriodStart, &r.CurrentPeriodEnd, &r.Retries, &r.NextRetryAt, &r.Currency, &r.Metadata,
		&r.CyclesProcessed, &r.TotalRevenue, &r.CancelledAt, &r.CreatedAt, &r.UpdatedAt,
	)
}

func (r subscriptionRow) toDomain() domain.Subscription {
	retries := 0
	if r.Retries != nil {
		retries = *r.Retries
	}
	return domain.Subscription{
		OrgId:              r.OrgId,
		Id:                 r.Id,
		PspId:              domain.Gateway(r.PspId),
		OrderId:            r.OrderId,
		CustomerId:         r.CustomerId,
		Status:             domain.SubscriptionStatus(r.Status),
		PaymentMethodId:    strOrEmpty(r.PaymentMethodId),
		StartDate:          timeOrZero(r.StartDate),
		EndDate:            timeOrZero(r.EndDate),
		BillingInterval:    domain.BillingInterval(r.BillingInterval),
		BillingIntervalQty: r.BillingIntervalQty,
		Cycles:             r.Cycles,
		BillingAnchor:      r.BillingAnchor,
		TrialInterval:      domain.BillingInterval(r.TrialInterval),
		TrialIntervalQty:   r.TrialIntervalQty,
		TrialEndsAt:        timeOrZero(r.TrialEndsAt),
		CancelAt:           timeOrZero(r.CancelAt),
		EndsAt:             timeOrZero(r.EndsAt),
		LastCharge:         timeOrZero(r.LastCharge),
		RenewsAt:           timeOrZero(r.RenewsAt),
		CurrentPeriodStart: timeOrZero(r.CurrentPeriodStart),
		CurrentPeriodEnd:   timeOrZero(r.CurrentPeriodEnd),
		Retries:            retries,
		NextRetryAt:        timeOrZero(r.NextRetryAt),
		Currency:           r.Currency,
		Metadata:           r.Metadata.V,
		CyclesProcessed:    r.CyclesProcessed,
		TotalRevenue:       r.TotalRevenue,
		CancelledAt:        timeOrZero(r.CancelledAt),
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

func subscriptionRowFromDomain(s domain.Subscription) subscriptionRow {
	retries := s.Retries
	return subscriptionRow{
		OrgId:              s.OrgId,
		Id:                 s.Id,
		PspId:              string(s.PspId),
		OrderId:            s.OrderId,
		CustomerId:         s.CustomerId,
		Status:             string(s.Status),
		PaymentMethodId:    nilIfEmpty(s.PaymentMethodId),
		StartDate:          nullTime(s.StartDate),
		EndDate:            nullTime(s.EndDate),
		BillingInterval:    string(s.BillingInterval),
		BillingIntervalQty: s.BillingIntervalQty,
		Cycles:             s.Cycles,
		BillingAnchor:      s.BillingAnchor,
		TrialInterval:      string(s.TrialInterval),
		TrialIntervalQty:   s.TrialIntervalQty,
		TrialEndsAt:        nullTime(s.TrialEndsAt),
		CancelAt:           nullTime(s.CancelAt),
		EndsAt:             nullTime(s.EndsAt),
		LastCharge:         nullTime(s.LastCharge),
		RenewsAt:           nullTime(s.RenewsAt),
		CurrentPeriodStart: nullTime(s.CurrentPeriodStart),
		CurrentPeriodEnd:   nullTime(s.CurrentPeriodEnd),
		Retries:            &retries,
		NextRetryAt:        nullTime(s.NextRetryAt),
		Currency:           s.Currency,
		Metadata:           newJSON(emptyIfNil(s.Metadata)),
		CyclesProcessed:    s.CyclesProcessed,
		TotalRevenue:       s.TotalRevenue,
		CancelledAt:        nullTime(s.CancelledAt),
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
}
