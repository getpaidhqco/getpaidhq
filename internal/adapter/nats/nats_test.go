package nats

import (
	"github.com/stretchr/testify/assert"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"
	"testing"
	"time"
)

func TestNatsPubSub_Publish(t *testing.T) {
	logger := lib.GetLogger()
	pubsub := NewNatsPubSub(logger)
	// Drain the conn and shut the embedded server down so the test doesn't leak
	// a goroutine + hold port 4222 for the rest of the process.
	t.Cleanup(func() { _ = pubsub.Close() })

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
