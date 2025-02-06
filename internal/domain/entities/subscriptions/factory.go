package subscriptions

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/lib"
	"time"
)

// NewSubscriptionFrominput creates a new Subscription from a payloop-cart input
func NewFromCreateInput(input CreateSubscriptionInput) entities.Subscription {

	var startDate = time.Now().UTC()
	var trialEndsAt *time.Time
	if input.TrialInterval != prices.BillingIntervalNone {
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

		trialEndsAt = &startDate
	}

	return entities.Subscription{
		OrgId:              input.OrgId,
		Id:                 lib.GenerateId("sub"),
		Status:             entities.SubscriptionStatusPending,
		StartDate:          startDate,
		EndDate:            nil,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		Cycles:             0,
		BillingAnchor:      startDate.Day(),
		TrialEndsAt:        trialEndsAt,
		CancelAt:           nil,
		EndsAt:             nil,
		LastCharge:         nil,
		RenewsAt:           nil,
		Retries:            0,
		NextRetry:          nil,
		Currency:           input.Currency,
		Amount:             input.Amount,
		Metadata:           nil,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CancelledAt:        nil,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}
