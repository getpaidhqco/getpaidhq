package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UserHandler is a placeholder today — no routes wired. The test pins that
// contract: NewUserHandler returns a non-nil handler, and RegisterRoutes is a
// no-op that does not panic, so a future wiring change is visible.
func TestUserHandler_PlaceholderContract(t *testing.T) {
	ts := newTestServer()
	h := NewUserHandler(nil, silentLogger{})
	require.NotNil(t, h)

	before := mustRouteCount(t, ts)
	h.RegisterRoutes(ts.srv)
	after := mustRouteCount(t, ts)

	assert.Equal(t, before, after, "UserHandler.RegisterRoutes must not register any routes")
}

// mustRouteCount returns the number of registered routes on the server. Used
// to assert that a no-op RegisterRoutes really registers nothing.
func mustRouteCount(t *testing.T, ts *testSrv) int {
	t.Helper()
	// fuego exposes the route table via the OpenAPI doc.
	return len(ts.srv.OpenAPI.Description().Paths.Map())
}
