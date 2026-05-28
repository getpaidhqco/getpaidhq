package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthHandler_Healthcheck(t *testing.T) {
	// Health needs no services and ignores authz/authn. A plain server with the
	// route registered is the minimum that proves the handler is wired.
	ts := newTestServer()
	NewHealthHandler(silentLogger{}).RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/health", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got HealthResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, "ok", got.Status)
}
