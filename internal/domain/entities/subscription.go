package entities

import (
	cart "github.com/mdwt/payloop-cart"
	"github.com/mdwt/payloop-cart/types"
	"payloop/internal/lib"
	"time"
)

type SubscriptionStatus string

const (
	SubscriptionStatusTrial     SubscriptionStatus = "trial"
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"
	SubscriptionStatusPaused    SubscriptionStatus = "paused"
	SubscriptionStatusUnpaid    SubscriptionStatus = "unpaid"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusPending   SubscriptionStatus = "pending"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
)

type Subscription struct {
	OrgId              string             `json:"org_id"`
	Id                 string             `json:"id"`
	OrderId            string             `json:"order_id"`
	Status             SubscriptionStatus `json:"status"`
	StartDate          time.Time          `json:"start_date"`
	EndDate            *time.Time         `json:"end_date"`
	BillingInterval    string             `json:"billing_interval"`
	BillingIntervalQty int                `json:"billing_interval_qty"`
	Cycles             int                `json:"cycles"`
	BillingAnchor      int                `json:"billing_anchor"`
	TrialEndsAt        *time.Time         `json:"trial_ends_at"`
	CancelAt           *time.Time         `json:"cancel_at"`
	EndsAt             *time.Time         `json:"ends_at"`
	LastCharge         *time.Time         `json:"last_charge"`
	RenewsAt           *time.Time         `json:"renews_at"`
	Retries            int                `json:"retries"`
	NextRetry          *time.Time         `json:"next_retry"`
	Currency           string             `json:"currency"`
	Amount             int                `json:"amount"`
	Metadata           map[string]string  `json:"metadata"`
	CyclesProcessed    int                `json:"cycles_processed"`
	TotalRevenue       int                `json:"total_revenue"`
	CancelledAt        *time.Time         `json:"cancelled_at"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// NewSubscriptionFromItem creates a new Subscription from a payloop-cart Item
func NewSubscriptionFromItem(orgId, orderId string, item cart.Item) Subscription {

	var startDate time.Time
	if item.Price.TrialInterval == types.BillingIntervalNone {
		startDate = time.Now()
	}

	return Subscription{
		OrgId:              orgId,
		Id:                 lib.GenerateId("subscription"),
		OrderId:            orderId,
		Status:             SubscriptionStatusPending,
		StartDate:          startDate,
		EndDate:            nil,
		BillingInterval:    string(item.Price.BillingInterval),
		BillingIntervalQty: item.Price.BillingIntervalQty,
		Cycles:             0,
		BillingAnchor:      startDate.Day(),
		TrialEndsAt:        nil,
		CancelAt:           nil,
		EndsAt:             nil,
		LastCharge:         nil,
		RenewsAt:           nil,
		Retries:            0,
		NextRetry:          nil,
		Currency:           item.Price.Currency,
		Amount:             item.Price.UnitPrice,
		Metadata:           nil,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CancelledAt:        nil,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}
