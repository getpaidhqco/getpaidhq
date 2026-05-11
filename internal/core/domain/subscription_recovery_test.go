package domain

import (
	"testing"
	"time"
)

func TestRetryPolicy_GetNextCharge(t *testing.T) {
	subscription := Subscription{
		RenewsAt: time.Now(),
		Retries:  0,
	}

	retryPolicy := RetryPolicy{
		RetryAttempts: 5,
		RetryInterval: RetryIntervalDay,
		RetryPeriod:   14,
		FailureAction: FailureActionCancel,
	}

	for i := range retryPolicy.RetryAttempts {
		nextCharge := retryPolicy.GetNextCharge(subscription)
		t.Logf("Next charge date for retry %d: %v", i+1, nextCharge)
		subscription.Retries++
	}
}
