package domain

import (
	"testing"

	"go.uber.org/goleak"
)

// Domain tests are pure (no goroutines), so any leak detected here is a
// regression introduced by new test setup or by domain code reaching into
// something that spawns goroutines. Keep this package goroutine-free.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
