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
	BillingInterval    BillingInterval    `json:"billing_interval"`
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

// NextBillingDate calculates and returns the next billing date for a subscription
// based on the StartDate, LastCharge, BillingInterval, BillingIntervalQty and CyclesProcessed
// If the subscription has not started yet, it returns the StartDate
// If the subscription has started but has not been charged yet, it returns the StartDate
// If the subscription has been charged, it uses the LastCharge date as the base date
// and the BillingInterval and BillingIntervalQty
func (s Subscription) NextBillingDate() time.Time {
	if s.BillingInterval == "" || s.BillingIntervalQty <= 0 {
		return time.Time{}
	}

	nextBillingDate := s.StartDate
	if s.LastCharge == nil && s.CyclesProcessed == 0 {
		return nextBillingDate
	}

	if s.LastCharge != nil && s.LastCharge.After(nextBillingDate) {
		nextBillingDate = *s.LastCharge
	}

	switch s.BillingInterval {
	case "minute":
		nextBillingDate = nextBillingDate.Add(time.Minute * time.Duration(s.BillingIntervalQty))
	case "hour":
		nextBillingDate = nextBillingDate.Add(time.Hour * time.Duration(s.BillingIntervalQty))
	case "day":
		nextBillingDate = nextBillingDate.AddDate(0, 0, s.BillingIntervalQty)
	case "week":
		nextBillingDate = nextBillingDate.AddDate(0, 0, s.BillingIntervalQty*7)
	case "month":
		nextBillingDate = nextBillingDate.AddDate(0, s.BillingIntervalQty, 0)
	case "year":
		nextBillingDate = nextBillingDate.AddDate(s.BillingIntervalQty, 0, 0)
	}

	return nextBillingDate
}

// NewSubscriptionFromItem creates a new Subscription from a payloop-cart Item
func NewSubscriptionFromItem(orgId, orderId string, item cart.Item) Subscription {

	var startDate = time.Now().UTC()
	if item.Price.TrialInterval != types.BillingIntervalNone {
		startDate = startDate.AddDate(0, 0, item.Price.TrialIntervalQty)
	}

	return Subscription{
		OrgId:              orgId,
		Id:                 lib.GenerateId("sub"),
		OrderId:            orderId,
		Status:             SubscriptionStatusPending,
		StartDate:          startDate,
		EndDate:            nil,
		BillingInterval:    BillingInterval(item.Price.BillingInterval),
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
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}
