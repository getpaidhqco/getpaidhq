package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// newPaymentMethodHandlerForTest builds the standalone PaymentMethodHandler on
// the real CustomerService (against repo fakes), so these tests exercise the
// production path end-to-end through the /api/payment-methods/{id} route.
func newPaymentMethodHandlerForTest(t *testing.T, pmRepo *fakePaymentMethodRepo) *PaymentMethodHandler {
	t.Helper()
	svc, err := service.NewCustomerService(&fakeCustomerRepo{}, pmRepo, newPubSub(), silentLogger{}, noopScheduler{})
	if err != nil {
		t.Fatalf("NewCustomerService: %v", err)
	}
	return NewPaymentMethodHandler(svc)
}

func TestPaymentMethodHandler_Get_HappyPath(t *testing.T) {
	pmRepo := &fakePaymentMethodRepo{byId: domain.PaymentMethod{Id: "pm_1", Name: "Visa ****4242"}}
	pmh := newPaymentMethodHandlerForTest(t, pmRepo)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	pmh.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/payment-methods/pm_1", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got domain.PaymentMethod
	decodeJSON(t, rec, &got)
	assert.Equal(t, "pm_1", got.Id)
	assert.Equal(t, "Visa ****4242", got.Name)
}

// TestPaymentMethodHandler_Get_OrgAndIdPlumbing pins the tenant-isolation
// contract: the org passed to the repo comes from the authenticated user, and
// the id comes from the URL path param — never from anything client-supplied in
// the body. This is the same "OrgId from auth, not request" guarantee the
// customer/cart handlers assert.
func TestPaymentMethodHandler_Get_OrgAndIdPlumbing(t *testing.T) {
	pmRepo := &fakePaymentMethodRepo{byId: domain.PaymentMethod{Id: "pm_42"}}
	pmh := newPaymentMethodHandlerForTest(t, pmRepo)

	ts := newTestServer(fixedAuthMiddleware(ownerUser())) // ownerUser ⇒ OrgId "org_1"
	pmh.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/payment-methods/pm_42", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, "org_1", pmRepo.lastFindOrg, "org must be taken from the auth user")
	assert.Equal(t, "pm_42", pmRepo.lastFindId, "id must be taken from the path param")
}

func TestPaymentMethodHandler_Get_NotFound(t *testing.T) {
	// Any repo failure is wrapped by CustomerService.GetPaymentMethod as a
	// NotFound CustomError, which the handler renders as the 404 envelope.
	pmRepo := &fakePaymentMethodRepo{byIdErr: errors.New("no such row")}
	pmh := newPaymentMethodHandlerForTest(t, pmRepo)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	pmh.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/payment-methods/pm_x", nil)

	assertErrorEnvelope(t, rec, http.StatusNotFound, string(lib.NotFoundError))
}

// TestPaymentMethodHandler_Get_AuthnGatedNotAuthzGated documents and pins an
// important security property: the payment-method read does NOT call the Cedar
// authorizer — it only requires a valid authenticated user. So every role
// (including support, which has no permit rule in policy.cedar and is therefore
// denied on most actions) can read a payment method within its own org. Reads
// are gated by authentication + org-scoping, not by role.
//
// If payment-method reads should ever become role-restricted, this test is the
// canary: tighten the handler to Enforce an action and update the expectations.
func TestPaymentMethodHandler_Get_AuthnGatedNotAuthzGated(t *testing.T) {
	roles := []struct {
		name string
		user port.AuthUser
	}{
		{"owner", ownerUser()},
		{"member", memberUser()},
		{"admin", adminUser()},
		{"support (no permit rule)", supportUser()},
	}

	for _, tc := range roles {
		t.Run(tc.name, func(t *testing.T) {
			pmRepo := &fakePaymentMethodRepo{byId: domain.PaymentMethod{Id: "pm_1"}}
			pmh := newPaymentMethodHandlerForTest(t, pmRepo)

			ts := newTestServer(fixedAuthMiddleware(tc.user))
			pmh.RegisterRoutes(ts.api())

			rec := doJSON(t, ts, http.MethodGet, "/api/payment-methods/pm_1", nil)

			require.Equal(t, http.StatusOK, rec.Code,
				"payment-method read is authn-gated only; %s should be allowed (body=%s)",
				tc.name, rec.Body.String())
		})
	}
}
