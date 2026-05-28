package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

func newPspHandlerForTest(
	t *testing.T,
	psp *fakePspRepo,
	setting *fakeSettingRepo,
) *PspHandler {
	t.Helper()
	svc := service.NewPspService(psp, setting, silentLogger{}, newPubSub())
	return NewPspHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestPspHandler_Create(t *testing.T) {
	t.Run("admin creates a gateway — psp + settings rows persisted", func(t *testing.T) {
		// CreatePaymentServiceProvider is not in policy.cedar's owner/member
		// permit lists; only the unconditional admin rule covers it.
		psp := &fakePspRepo{}
		setting := &fakeSettingRepo{}
		h := newPspHandlerForTest(t, psp, setting)

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/gateways", CreateGatewayRequest{
			Name: "Live Paystack", PspId: "paystack", Settings: map[string]string{"secret": "sk_x"},
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got GatewayResponse
		decodeJSON(t, rec, &got)
		assert.NotEmpty(t, got.Id)
		require.Len(t, psp.created, 1)
		require.Len(t, setting.created, 1, "settings row created alongside the psp config")
	})

	t.Run("non-admin role is denied by cedar", func(t *testing.T) {
		// owner is permitted only the actions explicitly listed in
		// policy.cedar; CreatePaymentServiceProvider is admin-only.
		h := newPspHandlerForTest(t, &fakePspRepo{}, &fakeSettingRepo{})

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/gateways", CreateGatewayRequest{
			Name: "Paystack", PspId: "paystack", Settings: map[string]string{},
		})

		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
	})
}
