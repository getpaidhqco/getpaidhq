package handler

import (
	"context"
	"getpaidhq/internal/lib/errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type fakePaymentRepoHTTP struct {
	port.PaymentRepository
	list []domain.Payment
	one  domain.Payment
}

func (r *fakePaymentRepoHTTP) List(_ context.Context, _ string, _ domain.Pagination) ([]domain.Payment, int, error) {
	return r.list, len(r.list), nil
}
func (r *fakePaymentRepoHTTP) FindById(_ context.Context, _, _ string) (domain.Payment, error) {
	return r.one, nil
}
func (r *fakePaymentRepoHTTP) FindBySubscriptionId(_ context.Context, _, _ string, _ domain.Pagination) ([]domain.Payment, int, error) {
	return r.list, len(r.list), nil
}

func newPaymentHandlerForTest(t *testing.T, repo *fakePaymentRepoHTTP) *PaymentHandler {
	t.Helper()
	return NewPaymentHandler(service.NewPaymentService(repo, silentLogger{}), silentLogger{}, newRealAuthz(t))
}

func TestPaymentHandler_AuthzGuards(t *testing.T) {
	h := newPaymentHandlerForTest(t, &fakePaymentRepoHTTP{})
	ts := newTestServer(fixedAuthMiddleware(supportUser()))
	h.RegisterRoutes(ts.api())
	for _, path := range []string{"/api/payments", "/api/payments/pay_1"} {
		rec := doJSON(t, ts, http.MethodGet, path, nil)
		assertErrorEnvelope(t, rec, http.StatusForbidden, string(errors.ForbiddenError))
	}
}

func TestPaymentHandler_ListAndGet(t *testing.T) {
	pay := domain.Payment{Id: "pay_1", SubscriptionId: "sub_1", InvoiceId: "inv_1", Status: domain.PaymentStatusSucceeded, Amount: 1030}
	repo := &fakePaymentRepoHTTP{list: []domain.Payment{pay}, one: pay}
	h := newPaymentHandlerForTest(t, repo)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/payments", nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var list ListResponse
	decodeJSON(t, rec, &list)
	assert.Equal(t, 1, list.Meta.Total)

	rec = doJSON(t, ts, http.MethodGet, "/api/payments/pay_1", nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got PaymentResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, "pay_1", got.Id)
	assert.Equal(t, "inv_1", got.InvoiceId)
}
