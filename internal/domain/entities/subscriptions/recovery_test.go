package subscriptions

import (
	"payloop/internal/domain/entities"
	"testing"
	"time"
)

func TestRetryPolicy_GetNextCharge(t *testing.T) {
	subscription := entities.Subscription{
		RenewsAt: time.Now(),
		Retries:  0,
	}

	retryPolicy := RetryPolicy{
		RetryAttempts: 5,
		RetryInterval: RetryIntervalDay,
		RetryPeriod:   14,
		FailureAction: FailureActionCancel,
	}

	for i := 0; i < retryPolicy.RetryAttempts; i++ {
		nextCharge := retryPolicy.GetNextCharge(subscription)
		//daysAdd := float64(subscription.Retries+1) * float64(retryPolicy.RetryPeriod) / float64(retryPolicy.RetryAttempts)
		//expectedNextCharge := subscription.RenewsAt.Add(time.Duration(daysAdd) * 24 * time.Hour)
		//if nextCharge != expectedNextCharge {
		//	t.Errorf("expected next charge date to be %v, got %v", expectedNextCharge, nextCharge)
		//	break
		//}
		t.Logf("Next charge date for retry %d: %v", i+1, nextCharge)
		subscription.Retries++
	}
}
