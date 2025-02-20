package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
)

type Subscription struct {
	OrgId              string      `json:"org_id"`
	Id                 string      `json:"id"`
	PspId              string      `json:"psp_id"`
	OrderId            string      `json:"order_id"`
	OrderItemId        string      `json:"order_item_id"`
	OrderItem          OrderItem   `json:"-"`
	CustomerId         string      `json:"customer_id"`
	Customer           Customer    `json:"-"`
	Status             string      `json:"status"`
	PaymentMethodId    pgtype.Text `json:"payment_method_id,omitempty"`
	StartDate          pgtype.Date `json:"start_date"`
	EndDate            pgtype.Date `json:"end_date"`
	BillingInterval    string      `json:"billing_interval"`
	BillingIntervalQty int         `json:"billing_interval_qty"`
	Cycles             int         `json:"cycles"`
	BillingAnchor      int         `json:"billing_anchor"`
	TrialEndsAt        pgtype.Date `json:"trial_ends_at"`
	CancelAt           pgtype.Date `json:"cancel_at"`
	EndsAt             pgtype.Date `json:"ends_at"`
	LastCharge         pgtype.Date `json:"last_charge"`
	RenewsAt           pgtype.Date `json:"renews_at"`

	CurrentPeriodStart pgtype.Date `json:"current_period_start"`
	CurrentPeriodEnd   pgtype.Date `json:"current_period_end"`

	Retries     int         `json:"retries"`
	NextRetryAt pgtype.Date `json:"next_retry"`

	Currency        string            `json:"currency"`
	Amount          int64             `json:"amount"`
	Metadata        map[string]string `json:"metadata"`
	CyclesProcessed int               `json:"cycles_processed"`
	TotalRevenue    int64             `json:"total_revenue"`
	CancelledAt     pgtype.Date       `json:"cancelled_at"`
	CreatedAt       pgtype.Date       `json:"created_at"`
	UpdatedAt       pgtype.Date       `json:"updated_at"`
}

func (s *Subscription) ToEntity() entities.Subscription {
	return entities.Subscription{
		OrgId:              s.OrgId,
		Id:                 s.Id,
		PspId:              common.Gateway(s.PspId),
		OrderId:            s.OrderId,
		OrderItemId:        s.OrderItemId,
		OrderItem:          s.OrderItem.ToEntity(),
		CustomerId:         s.CustomerId,
		Customer:           s.Customer.ToEntity(),
		Status:             entities.SubscriptionStatus(s.Status),
		PaymentMethodId:    s.PaymentMethodId.String,
		StartDate:          s.StartDate.Time,
		EndDate:            s.EndDate.Time,
		BillingInterval:    prices.BillingInterval(s.BillingInterval),
		BillingIntervalQty: s.BillingIntervalQty,
		Cycles:             s.Cycles,
		BillingAnchor:      s.BillingAnchor,
		TrialEndsAt:        s.TrialEndsAt.Time,
		CancelAt:           s.CancelAt.Time,
		EndsAt:             s.EndsAt.Time,
		LastCharge:         s.LastCharge.Time,
		RenewsAt:           s.RenewsAt.Time,
		CurrentPeriodStart: s.CurrentPeriodStart.Time,
		CurrentPeriodEnd:   s.CurrentPeriodEnd.Time,
		Retries:            s.Retries,
		NextRetryAt:        s.NextRetryAt.Time,
		Currency:           s.Currency,
		Amount:             s.Amount,
		Metadata:           s.Metadata,
		CyclesProcessed:    s.CyclesProcessed,
		TotalRevenue:       s.TotalRevenue,
		CancelledAt:        s.CancelledAt.Time,
		CreatedAt:          s.CreatedAt.Time,
		UpdatedAt:          s.UpdatedAt.Time,
	}
}
