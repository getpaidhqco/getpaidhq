package nats

import (
	"github.com/stretchr/testify/assert"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
	"testing"
	"time"
)

func TestNatsPubSub_Publish(t *testing.T) {
	logger := lib.GetLogger()
	pubsub := NewNatsPubSub(logger)

	err := pubsub.PublishJSON("subscription.paused", entities.Subscription{
		OrgId:              "mollie",
		Id:                 "sub_2saZn2yvjfnzJ6Io2yfgEsCwtmg",
		OrderId:            "",
		Status:             "paused",
		StartDate:          time.Time{},
		EndDate:            nil,
		BillingInterval:    "",
		BillingIntervalQty: 0,
		Cycles:             0,
		BillingAnchor:      0,
		TrialEndsAt:        nil,
		CancelAt:           nil,
		EndsAt:             nil,
		LastCharge:         nil,
		RenewsAt:           nil,
		Retries:            0,
		NextRetry:          nil,
		Currency:           "",
		Amount:             0,
		Metadata:           nil,
		CyclesProcessed:    0,
		TotalRevenue:       0,
		CancelledAt:        nil,
		CreatedAt:          time.Time{},
		UpdatedAt:          time.Time{},
	})
	assert.NoError(t, err)
}
