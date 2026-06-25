package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

type fakeInvoiceRepoHTTP struct {
	port.InvoiceRepository
	list []domain.Invoice
	one  domain.Invoice
}

func (r *fakeInvoiceRepoHTTP) List(_ context.Context, _ string, _ domain.Pagination) ([]domain.Invoice, int, error) {
	return r.list, len(r.list), nil
}
func (r *fakeInvoiceRepoHTTP) FindById(_ context.Context, _, _ string) (domain.Invoice, error) {
	return r.one, nil
}
func (r *fakeInvoiceRepoHTTP) FindBySubscriptionId(_ context.Context, _, _ string, _ domain.Pagination) ([]domain.Invoice, int, error) {
	return r.list, len(r.list), nil
}

func newInvoiceHandlerForTest(t *testing.T, repo *fakeInvoiceRepoHTTP) *InvoiceHandler {
	t.Helper()
	svc := service.NewInvoiceService(repo, nil, nil, nil, nil, silentLogger{}, nil, nil, nil)
	return NewInvoiceHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestInvoiceHandler_AuthzGuards(t *testing.T) {
	h := newInvoiceHandlerForTest(t, &fakeInvoiceRepoHTTP{})
	ts := newTestServer(fixedAuthMiddleware(supportUser()))
	h.RegisterRoutes(ts.api())
	for _, path := range []string{"/api/invoices", "/api/invoices/inv_1", "/api/subscriptions/sub_1/invoices"} {
		rec := doJSON(t, ts, http.MethodGet, path, nil)
		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
	}
}

func TestInvoiceHandler_ListAndGet(t *testing.T) {
	inv := domain.Invoice{
		Id: "inv_1", SubscriptionId: "sub_1", Status: domain.InvoiceStatusPaid, Total: 1030,
		LineItems: []domain.InvoiceLineItem{
			{Id: "ili_1", Kind: domain.InvoiceLineKindBase, Quantity: decimal.NewFromInt(1), UnitAmount: decimal.NewFromInt(1000), Total: 1000},
			{Id: "ili_2", Kind: domain.InvoiceLineKindUsage, Quantity: decimal.NewFromInt(3), UnitAmount: decimal.NewFromInt(10), Total: 30},
		},
	}
	repo := &fakeInvoiceRepoHTTP{list: []domain.Invoice{inv}, one: inv}
	h := newInvoiceHandlerForTest(t, repo)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/invoices", nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var list ListResponse
	decodeJSON(t, rec, &list)
	assert.Equal(t, 1, list.Meta.Total)

	rec = doJSON(t, ts, http.MethodGet, "/api/invoices/inv_1", nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got InvoiceResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, "inv_1", got.Id)
	assert.Len(t, got.LineItems, 2)
	assert.Equal(t, int64(1030), got.Total)
}
