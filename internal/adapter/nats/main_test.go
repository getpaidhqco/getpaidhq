package nats

import (
	"testing"

	"go.uber.org/goleak"
)

// Guards that NewNatsPubSub's embedded server + client are fully torn down by
// Close() — if a future change drops the Drain/Shutdown, goleak fails here.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
