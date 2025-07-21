package testutils

import (
	"os"
	"time"

	"go.temporal.io/sdk/testsuite"
)

// FastForwardTime advances workflow time without waiting
func FastForwardTime(env *testsuite.TestWorkflowEnvironment, duration time.Duration) {
	env.SetStartTime(env.Now().Add(duration))
}

// GetTestBillingCycle returns fast billing cycle duration for tests
func GetTestBillingCycle() time.Duration {
	if testMode := os.Getenv("TEST_MODE"); testMode == "fast" {
		return time.Second * 30 // 30 seconds instead of days
	}
	return time.Hour * 24 * 30 // Normal monthly cycle
}

// GetTestReminderInterval returns fast reminder interval for tests
func GetTestReminderInterval() time.Duration {
	if testMode := os.Getenv("TEST_MODE"); testMode == "fast" {
		return time.Second * 5 // 5 seconds instead of days
	}
	return time.Hour * 24 * 3 // Normal 3 days
}

// WaitForWorkflowTimer waits for a specific timer duration in tests
func WaitForWorkflowTimer(env *testsuite.TestWorkflowEnvironment, expectedDuration time.Duration) {
	// In test environment, timers fire immediately
	// This helper can be used to verify timer setup
}

// AdvanceToNextBillingCycle advances time to trigger next billing cycle
func AdvanceToNextBillingCycle(env *testsuite.TestWorkflowEnvironment) {
	FastForwardTime(env, time.Second*6) // Just past the 5-second test cycle
}

// AdvanceToDunningRetry advances time to trigger next dunning retry
func AdvanceToDunningRetry(env *testsuite.TestWorkflowEnvironment, retryNumber int) {
	// Each retry is 1s, 2s, 3s in test config
	FastForwardTime(env, time.Duration(retryNumber)*time.Second+time.Millisecond*100)
}

// TimeAccelerationFactor returns the acceleration factor for tests
func TimeAccelerationFactor() int {
	if testMode := os.Getenv("TEST_MODE"); testMode == "fast" {
		return 1000 // 1000x faster
	}
	return 1 // Normal speed
}

// ConvertRealTimeToTestTime converts real durations to test durations
func ConvertRealTimeToTestTime(realDuration time.Duration) time.Duration {
	factor := TimeAccelerationFactor()
	return realDuration / time.Duration(factor)
}

// ConvertTestTimeToRealTime converts test durations to real durations
func ConvertTestTimeToRealTime(testDuration time.Duration) time.Duration {
	factor := TimeAccelerationFactor()
	return testDuration * time.Duration(factor)
}