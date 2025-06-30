package response

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
	"time"
)

type Subscription struct {
	Id          string                      `json:"id"`
	Status      entities.SubscriptionStatus `json:"status"`
	Currency    string                      `json:"currency"`
	Amount      int64                       `json:"amount"`
	OrderId     string                      `json:"order_id"`
	OrderItemId string                      `json:"order_item_id"`

	PaymentMethodId string `json:"payment_method_id,omitempty"`

	StartDate          time.Time              `json:"start_date,omitempty"`
	EndDate            time.Time              `json:"end_date,omitempty,omitzero"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	Cycles             int                    `json:"cycles"`
	BillingAnchor      int                    `json:"billing_anchor"`
	TrialEndsAt        time.Time              `json:"trial_ends_at,omitempty,omitzero"`
	CancelAt           time.Time              `json:"cancel_at,omitempty,omitzero"`
	EndsAt             time.Time              `json:"ends_at,omitempty,omitzero"`
	LastCharge         time.Time              `json:"last_charge,omitempty,omitzero"`
	RenewsAt           time.Time              `json:"renews_at,omitempty,omitzero"`

	CurrentPeriodStart time.Time `json:"current_period_start,omitempty,omitzero"`
	CurrentPeriodEnd   time.Time `json:"current_period_end,omitempty,omitzero"`

	Retries     int       `json:"retries"`
	NextRetryAt time.Time `json:"next_retry,omitempty,omitzero"`

	Customer Customer `json:"customer"`

	Metadata        map[string]string `json:"metadata"`
	CyclesProcessed int               `json:"cycles_processed"`
	TotalRevenue    int64             `json:"total_revenue"`
	CancelledAt     time.Time         `json:"cancelled_at,omitempty,omitzero"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func NewSubscriptionFromEntity(entity entities.Subscription) Subscription {
	return Subscription{
		Id:                 entity.Id,
		OrderId:            entity.OrderId,
		OrderItemId:        entity.OrderItemId,
		Customer:           NewCustomerFromEntity(entity.Customer),
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
