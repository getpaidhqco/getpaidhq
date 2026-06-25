package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

func newInvoiceSettingsHandlerForTest(t *testing.T, repo *fakeSettingRepoHTTP) *InvoiceSettingsHandler {
	t.Helper()
	svc := service.NewInvoiceSettingsService(repo, silentLogger{})
	return NewInvoiceSettingsHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestInvoiceSettingsHandler_Get_DefaultWhenUnset(t *testing.T) {
	// With nothing stored, ResolveInvoiceSettings returns DefaultInvoiceSettings
	// ("INV-" prefix, 6-digit padding).
	h := newInvoiceSettingsHandlerForTest(t, newFakeSettingRepoHTTP())

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/billing/invoice-settings", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got InvoiceSettingsDTO
	decodeJSON(t, rec, &got)
	require.Equal(t, "INV-", got.Prefix)
	require.Equal(t, 6, got.Padding)
}

func TestInvoiceSettingsHandler_Put_Then_Get_RoundTrips(t *testing.T) {
	h := newInvoiceSettingsHandlerForTest(t, newFakeSettingRepoHTTP())

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	// PUT returns the saved value.
	rec := doJSON(t, ts, http.MethodPut, "/api/billing/invoice-settings", InvoiceSettingsDTO{
		Prefix:  "ACME-",
		Padding: 4,
	})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var put InvoiceSettingsDTO
	decodeJSON(t, rec, &put)
	require.Equal(t, "ACME-", put.Prefix)
	require.Equal(t, 4, put.Padding)

	// GET reads it back.
	rec = doJSON(t, ts, http.MethodGet, "/api/billing/invoice-settings", nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got InvoiceSettingsDTO
	decodeJSON(t, rec, &got)
	require.Equal(t, "ACME-", got.Prefix)
	require.Equal(t, 4, got.Padding)
}

func TestInvoiceSettingsHandler_AuthzGuards(t *testing.T) {
	// support role is granted none of the settings actions → 403 on both verbs.
	repo := newFakeSettingRepoHTTP()
	h := newInvoiceSettingsHandlerForTest(t, repo)

	ts := newTestServer(fixedAuthMiddleware(supportUser()))
	h.RegisterRoutes(ts.api())

	cases := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/billing/invoice-settings", nil},
		{http.MethodPut, "/api/billing/invoice-settings", InvoiceSettingsDTO{Prefix: "X-", Padding: 3}},
	}
	for _, tc := range cases {
		rec := doJSON(t, ts, tc.method, tc.path, tc.body)
		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
	}
	require.Zero(t, repo.upsertN)
}
