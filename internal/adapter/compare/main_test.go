package compare

import (
	"testing"

	"go.uber.org/goleak"
)

// The compare EventStore spawns a background goroutine per read to check the
// secondary backend. Each is bounded by a timeout and a semaphore, so the package
// must end leak-free — goleak guards that the background checks always terminate
// (and that the semaphore is always released).
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
