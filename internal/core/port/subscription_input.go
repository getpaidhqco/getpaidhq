package port

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"
	"time"
)

// CreateSubscriptionInput is the input for SubscriptionService.Create.
type CreateSubscriptionInput struct {
	OrgId              string
	PaymentMethodId    string
	Activate           bool
	Amount             int64
	Currency           string
	BillingInterval    domain.BillingInterval
	BillingIntervalQty int
	Cycles             int
	TrialInterval      domain.BillingInterval
	TrialIntervalQty   int
	Metadata           map[string]string
}

// ToSubscription constructs a domain.Subscription from the input. Replaces the
// old input.ToSubscription() factory, which would have required
// domain to import service.
func (input CreateSubscriptionInput) ToSubscription() domain.Subscription {
	var startDate = time.Now().UTC()
	var trialEndsAt time.Time
	if input.TrialInterval != domain.BillingIntervalNone {
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

	return domain.Subscription{
		OrgId:              input.OrgId,
		Id:                 lib.GenerateId("sub"),
		Status:             domain.SubscriptionStatusPending,
		StartDate:          startDate,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		Cycles:             0,
		BillingAnchor:      startDate.Day(),
		TrialEndsAt:        trialEndsAt,
		Retries:            0,
		Currency:           input.Currency,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

// UpdateSubscriptionInput is the input for SubscriptionService.Update.
type UpdateSubscriptionInput struct {
	OrgId                string
	Id                   string
	Status               domain.SubscriptionStatus
	DefaultPaymentMethod string
	Metadata             map[string]string
}

// PauseSubscriptionInput is the input for SubscriptionService.Pause.
type PauseSubscriptionInput struct {
	OrgId  string
	Id     string
	Reason string
}

// ResumeSubscriptionInput is the input for SubscriptionService.Resume.
type ResumeSubscriptionInput struct {
	OrgId          string
	Id             string
	ResumeBehavior domain.SubscriptionResumeBehavior
}

// OutstandingInvoiceAction decides what happens to a still-open invoice when a
// subscription is voluntarily cancelled. Empty defaults to uncollectible.
type OutstandingInvoiceAction string

const (
	OutstandingInvoiceUncollectible OutstandingInvoiceAction = "uncollectible"
	OutstandingInvoiceVoid          OutstandingInvoiceAction = "void"
	OutstandingInvoiceKeep          OutstandingInvoiceAction = "keep"
)

// CancelSubscriptionInput is the input for SubscriptionService.Cancel.
type CancelSubscriptionInput struct {
	OrgId              string
	Id                 string
	Reason             string
	OutstandingInvoice OutstandingInvoiceAction // empty => uncollectible
}

// UpdateBillingAnchorInput is the input for SubscriptionService.UpdateBillingAnchor.
type UpdateBillingAnchorInput struct {
	OrgId         string
	Id            string
	BillingAnchor int
	ProrationMode domain.ProrationMode
}
