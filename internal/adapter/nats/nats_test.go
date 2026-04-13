package nats

import (
	"github.com/stretchr/testify/assert"
	"payloop/internal/core/domain"
	"payloop/internal/lib"
	"testing"
	"time"
)

func TestNatsPubSub_Publish(t *testing.T) {
	logger := lib.GetLogger()
	pubsub := NewNatsPubSub(logger)

	err := pubsub.Publish("mollie", "subscription.paused", domain.Subscription{
		OrgId:              "mollie",
		Id:                 "sub_2saZn2yvjfnzJ6Io2yfgEsCwtmg",
		OrderId:            "",
		Status:             "paused",
		StartDate:          time.Time{},
		EndDate:            time.Time{},
		BillingInterval:    "",
		BillingIntervalQty: 0,
		Cycles:             0,
		BillingAnchor:      0,
		TrialEndsAt:        time.Time{},
		CancelAt:           time.Time{},
		EndsAt:             time.Time{},
		LastCharge:         time.Time{},
		RenewsAt:           time.Time{},
		Retries:            0,
		NextRetryAt:        time.Time{},
		Currency:           "",
		Amount:             0,
		Metadata:           nil,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CancelledAt:        time.Time{},
		CreatedAt:          time.Time{},
		UpdatedAt:          time.Time{},
	})
	assert.NoError(t, err)
}
