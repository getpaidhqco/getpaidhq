package service

import (
	"testing"

	"go.uber.org/goleak"
)

// The service tests drive services through hand-rolled fakes that spawn no
// goroutines, so this package should stay leak-free. goleak guards that as the
// suite grows toward real engine/pubsub doubles.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
