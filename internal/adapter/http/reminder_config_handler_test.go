package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

func newReminderConfigHandlerForTest(t *testing.T, setting port.SettingRepository) *ReminderConfigHandler {
	t.Helper()
	svc := service.NewReminderConfigService(setting, silentLogger{})
	return NewReminderConfigHandler(svc, silentLogger{})
}

func TestReminderConfigHandler_Get_DefaultWhenUnset(t *testing.T) {
	// An empty setting Value parses to DefaultReminderConfig (one 7d/168h offset).
	h := newReminderConfigHandlerForTest(t, &fakeSettingRepo{})

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/billing/reminder-config", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ReminderConfigDTO
	decodeJSON(t, rec, &got)
	require.True(t, got.Enabled)
	require.Equal(t, []string{"168h0m0s"}, got.Offsets)
}

func TestReminderConfigHandler_Put_InvalidOffsetIsBadRequest(t *testing.T) {
	h := newReminderConfigHandlerForTest(t, &fakeSettingRepo{})

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPut, "/api/billing/reminder-config", ReminderConfigDTO{
		Enabled: true,
		Offsets: []string{"not-a-duration"},
	})

	require.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
}
