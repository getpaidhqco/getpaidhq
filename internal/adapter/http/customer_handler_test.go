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

// newCustomerHandlerForTest constructs a real CustomerService against repo
// fakes plus a real authz so we exercise the cedar policy. The returned setup
// can be wired into any test server.
func newCustomerHandlerForTest(
	t *testing.T,
	custRepo *fakeCustomerRepo,
	pmRepo *fakePaymentMethodRepo,
	ps *recordingPubSub,
) *CustomerHandler {
	t.Helper()
	svc, err := service.NewCustomerService(custRepo, pmRepo, ps, silentLogger{}, noopScheduler{})
	if err != nil {
		t.Fatalf("NewCustomerService: %v", err)
	}
	return NewCustomerHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestCustomerHandler_Create(t *testing.T) {
	// CustomerHandler.Create now enforces ActionCreateCustomer. The Cedar
	// policy has no owner/member permit rule for that action, so it's
	// effectively admin-only via the catch-all admin policy. Tests drive
	// adminUser() to exercise the happy path and ownerUser() to pin the
	// denial.
	t.Run("happy path persists + publishes + returns the customer", func(t *testing.T) {
		ps := newPubSub()
		custRepo := &fakeCustomerRepo{}
		h := newCustomerHandlerForTest(t, custRepo, &fakePaymentMethodRepo{}, ps)

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/customers", port.CreateCustomerInput{
			Email:     "jane@example.com",
			FirstName: "Jane",
			LastName:  "Doe",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got domain.Customer
		decodeJSON(t, rec, &got)
		assert.Equal(t, "jane@example.com", got.Email)
		require.Len(t, custRepo.created, 1)
		// Service takes OrgId from the auth user, not the request body — the
		// row written to the repo proves that.
		assert.Equal(t, "org_1", custRepo.created[0].OrgId)
		// At least one publish should have fired for customer.created.
		assert.Contains(t, ps.topicsPublished(), "customer.created")
	})

	t.Run("non-admin (owner) is denied — admin-only by virtue of no permit rule", func(t *testing.T) {
		custRepo := &fakeCustomerRepo{}
		h := newCustomerHandlerForTest(t, custRepo, &fakePaymentMethodRepo{}, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/customers", port.CreateCustomerInput{
			Email: "x@example.com",
		})

		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
		assert.Empty(t, custRepo.created, "service must not run when authz denies")
	})

	t.Run("duplicate email is surfaced as bad_request envelope", func(t *testing.T) {
		// Existing customer hit on FindByEmail => CustomerService returns a
		// BadRequest CustomError, which the handler should marshal to a 400.
		custRepo := &fakeCustomerRepo{byEmail: domain.Customer{Id: "existing"}}
		h := newCustomerHandlerForTest(t, custRepo, &fakePaymentMethodRepo{}, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/customers", port.CreateCustomerInput{
			Email: "dup@example.com",
		})

		assertErrorEnvelope(t, rec, http.StatusBadRequest, string(lib.BadRequestError))
	})

	t.Run("repo lookup failure becomes internal_error envelope", func(t *testing.T) {
		// FindByEmail errors => CustomerService wraps as InternalError CustomError.
		custRepo := &fakeCustomerRepo{byEmailErr: errors.New("db unreachable")}
		h := newCustomerHandlerForTest(t, custRepo, &fakePaymentMethodRepo{}, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/customers", port.CreateCustomerInput{
			Email: "x@example.com",
		})

		assertErrorEnvelope(t, rec, http.StatusInternalServerError, string(lib.InternalError))
	})
}

func TestCustomerHandler_Get(t *testing.T) {
	t.Run("returns the customer marshalled in the response shape", func(t *testing.T) {
		custRepo := &fakeCustomerRepo{byId: domain.Customer{
			OrgId: "org_1", Id: "cus_1", Email: "a@b.com", FirstName: "A",
		}}
		h := newCustomerHandlerForTest(t, custRepo, &fakePaymentMethodRepo{}, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/customers/cus_1", nil)

		require.Equal(t, http.StatusOK, rec.Code)
		var got CustomerResponse
		decodeJSON(t, rec, &got)
		assert.Equal(t, "cus_1", got.Id)
		assert.Equal(t, "a@b.com", got.Email)
		assert.Equal(t, "A", got.FirstName)
	})

	t.Run("repo error maps to not_found envelope", func(t *testing.T) {
		// CustomerService.Get wraps the underlying repo failure as a NotFound
		// CustomError; the handler propagates that into the ApiError envelope.
		custRepo := &fakeCustomerRepo{byIdErr: errors.New("nope")}
		h := newCustomerHandlerForTest(t, custRepo, &fakePaymentMethodRepo{}, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/customers/cus_x", nil)

		assertErrorEnvelope(t, rec, http.StatusNotFound, string(lib.NotFoundError))
	})
}

func TestCustomerHandler_List(t *testing.T) {
	custRepo := &fakeCustomerRepo{
		listResult: []domain.Customer{
			{Id: "cus_1", Email: "a@b.com"},
			{Id: "cus_2", Email: "c@d.com"},
		},
	}
	h := newCustomerHandlerForTest(t, custRepo, &fakePaymentMethodRepo{}, newPubSub())

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/customers?page=0&limit=10", nil)

	require.Equal(t, http.StatusOK, rec.Code)
	var got ListResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 2, got.Meta.Total)
	assert.Equal(t, 10, got.Meta.Limit)
}

func TestCustomerHandler_CreatePaymentMethod(t *testing.T) {
	// Also admin-only via no-permit-rule, same as Create above.
	t.Run("happy path", func(t *testing.T) {
		custRepo := &fakeCustomerRepo{byId: domain.Customer{
			OrgId: "org_1", Id: "cus_1",
			BillingAddress: domain.Address{Line1: "1 St", City: "London", Country: domain.Country("GB")},
		}}
		pmRepo := &fakePaymentMethodRepo{}
		h := newCustomerHandlerForTest(t, custRepo, pmRepo, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/customers/cus_1/payment-methods", port.CreatePaymentMethodInput{
			Psp:   "paystack",
			Name:  "Visa ****",
			Type:  domain.PaymentMethodType("card"),
			Token: "tok_visa",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		require.Len(t, pmRepo.created, 1)
		// CustomerId is taken from the path param, not the body.
		assert.Equal(t, "cus_1", pmRepo.created[0].CustomerId)
		// The token is accepted inbound and stored, but never echoed back.
		assert.Equal(t, "tok_visa", pmRepo.created[0].Token.Reveal())
		assert.NotContains(t, rec.Body.String(), "tok_visa")
	})

	t.Run("non-admin (owner) is denied", func(t *testing.T) {
		pmRepo := &fakePaymentMethodRepo{}
		h := newCustomerHandlerForTest(t, &fakeCustomerRepo{}, pmRepo, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/customers/cus_1/payment-methods", port.CreatePaymentMethodInput{
			Psp: "paystack", Name: "Visa", Type: domain.PaymentMethodType("card"), Token: "tok",
		})

		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
		assert.Empty(t, pmRepo.created)
	})

	t.Run("missing customer surfaces NotFound envelope", func(t *testing.T) {
		custRepo := &fakeCustomerRepo{byIdErr: errors.New("nope")}
		h := newCustomerHandlerForTest(t, custRepo, &fakePaymentMethodRepo{}, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/customers/cus_x/payment-methods", port.CreatePaymentMethodInput{
			Psp: "paystack", Name: "Visa", Type: domain.PaymentMethodType("card"), Token: "tok",
		})

		assertErrorEnvelope(t, rec, http.StatusNotFound, string(lib.NotFoundError))
	})
}

func TestCustomerHandler_UpdatePaymentMethod(t *testing.T) {
	t.Run("happy path (admin)", func(t *testing.T) {
		custRepo := &fakeCustomerRepo{byId: domain.Customer{
			OrgId: "org_1", Id: "cus_1",
			BillingAddress: domain.Address{Line1: "1 St", City: "London", Country: domain.Country("GB")},
		}}
		pmRepo := &fakePaymentMethodRepo{byId: domain.PaymentMethod{Id: "pm_1", CustomerId: "cus_1"}}
		h := newCustomerHandlerForTest(t, custRepo, pmRepo, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPut, "/api/customers/cus_1/payment-methods/pm_1", port.UpdatePaymentMethodInput{
			Token: "tok_new",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		require.Len(t, pmRepo.updated, 1)
		assert.Equal(t, "tok_new", pmRepo.updated[0].Token.Reveal(), "the new token is persisted")
	})

	t.Run("non-admin (owner) is denied", func(t *testing.T) {
		pmRepo := &fakePaymentMethodRepo{byId: domain.PaymentMethod{Id: "pm_1"}}
		h := newCustomerHandlerForTest(t, &fakeCustomerRepo{}, pmRepo, newPubSub())

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPut, "/api/customers/cus_1/payment-methods/pm_1", port.UpdatePaymentMethodInput{
			Token: "tok_new",
		})

		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
		assert.Empty(t, pmRepo.updated)
	})
}
