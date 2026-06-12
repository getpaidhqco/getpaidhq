package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

func newPspHandlerForTest(
	t *testing.T,
	psp *fakePspRepo,
) *PspHandler {
	t.Helper()
	svc := service.NewPspService(psp, fakeSecretCipher{}, silentLogger{}, newPubSub())
	return NewPspHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestPspHandler_Create(t *testing.T) {
	t.Run("admin creates a gateway — credentials sealed, config readable, secret never echoed", func(t *testing.T) {
		// CreatePaymentServiceProvider is not in policy.cedar's owner/member
		// permit lists; only the unconditional admin rule covers it.
		psp := &fakePspRepo{}
		h := newPspHandlerForTest(t, psp)

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/gateways", CreateGatewayRequest{
			Name: "Live Paystack", PspId: "paystack",
			Config:      map[string]string{"connect_id": "cn_1"},
			Credentials: map[string]domain.Secret{"api_key": "sk_x"},
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got GatewayResponse
		decodeJSON(t, rec, &got)
		assert.NotEmpty(t, got.Id)
		assert.Equal(t, map[string]string{"connect_id": "cn_1"}, got.Config, "non-secret config is echoed")

		// The raw secret must not appear anywhere in the response, and the
		// stored row carries it only inside the sealed envelope.
		assert.NotContains(t, rec.Body.String(), "sk_x", "secret never appears in a response body")
		require.Len(t, psp.created, 1)
		row := psp.created[0]
		assert.True(t, strings.HasPrefix(row.EncryptedCredentials, "enc["), "credentials stored sealed")
		assert.NotContains(t, row.Config, "api_key", "secret not mis-stored in readable config")
	})

	t.Run("missing credentials is a validation error", func(t *testing.T) {
		psp := &fakePspRepo{}
		h := newPspHandlerForTest(t, psp)

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/gateways", CreateGatewayRequest{
			Name: "Live Paystack", PspId: "paystack",
			Config: map[string]string{"connect_id": "cn_1"},
		})

		assert.GreaterOrEqual(t, rec.Code, 400, "body=%s", rec.Body.String())
		assert.Empty(t, psp.created)
	})

	t.Run("non-admin role is denied by cedar", func(t *testing.T) {
		// owner is permitted only the actions explicitly listed in
		// policy.cedar; CreatePaymentServiceProvider is admin-only.
		h := newPspHandlerForTest(t, &fakePspRepo{})

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/gateways", CreateGatewayRequest{
			Name: "Paystack", PspId: "paystack",
			Credentials: map[string]domain.Secret{"api_key": "sk_x"},
		})

		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
	})
}
