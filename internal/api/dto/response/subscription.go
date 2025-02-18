package response

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
	"time"
)

type Subscription struct {
	Id                 string                      `json:"id"`
	OrderId            string                      `json:"order_id"`
	OrderItemId        string                      `json:"order_item_id"`
	Customer           Customer                    `json:"customer"`
	Status             entities.SubscriptionStatus `json:"status"`
	PaymentMethodId    *string                     `json:"payment_method_id,omitempty"`
	StartDate          time.Time                   `json:"start_date,omitempty"`
	EndDate            *time.Time                  `json:"end_date,omitempty"`
	BillingInterval    prices.BillingInterval      `json:"billing_interval"`
	BillingIntervalQty int                         `json:"billing_interval_qty"`
	Cycles             int                         `json:"cycles"`
	BillingAnchor      int                         `json:"billing_anchor"`
	TrialEndsAt        *time.Time                  `json:"trial_ends_at,omitempty"`
	CancelAt           *time.Time                  `json:"cancel_at,omitempty"`
	EndsAt             *time.Time                  `json:"ends_at,omitempty"`
	LastCharge         *time.Time                  `json:"last_charge,omitempty"`
	RenewsAt           *time.Time                  `json:"renews_at,omitempty"`

	CurrentPeriodStart time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   time.Time `json:"current_period_end,omitempty"`

	Retries     int        `json:"retries"`
	NextRetryAt *time.Time `json:"next_retry,omitempty"`

	Currency        string            `json:"currency"`
	Amount          int               `json:"amount"`
	Metadata        map[string]string `json:"metadata"`
	CyclesProcessed int               `json:"cycles_processed"`
	TotalRevenue    int               `json:"total_revenue"`
	CancelledAt     *time.Time        `json:"cancelled_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func NewFromEntity(entity entities.Subscription) Subscription {
	return Subscription{
		Id:                 entity.Id,
		OrderId:            entity.OrderId,
		OrderItemId:        entity.OrderItemId,
		Customer:           NewFromEntityCustomer(entity.Customer),
		Status:             entity.Status,
		PaymentMethodId:    entity.PaymentMethodId,
		StartDate:          entity.StartDate,
		EndDate:            entity.EndDate,
		BillingInterval:    entity.BillingInterval,
		BillingIntervalQty: entity.BillingIntervalQty,
		Cycles:             entity.Cycles,
		BillingAnchor:      entity.BillingAnchor,
		TrialEndsAt:        entity.TrialEndsAt,
		CancelAt:           entity.CancelAt,
		EndsAt:             entity.EndsAt,
		LastCharge:         entity.LastCharge,
		RenewsAt:           entity.RenewsAt,
		CurrentPeriodStart: entity.CurrentPeriodStart,
		CurrentPeriodEnd:   entity.CurrentPeriodEnd,
		Retries:            entity.Retries,
		NextRetryAt:        entity.NextRetryAt,
		Currency:           entity.Currency,
		Amount:             entity.Amount,
		Metadata:           entity.Metadata,
		CyclesProcessed:    entity.CyclesProcessed,
		TotalRevenue:       entity.TotalRevenue,
		CancelledAt:        entity.CancelledAt,
		CreatedAt:          entity.CreatedAt,
		UpdatedAt:          entity.UpdatedAt,
	}
}
