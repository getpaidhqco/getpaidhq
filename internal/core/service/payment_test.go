package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// fakeRefundPaymentRepo serves a payment by psp id and records the
// update/refund writes ProcessRefund performs.
type fakeRefundPaymentRepo struct {
	port.PaymentRepository
	byPspId     domain.Payment
	byPspIdErr  error
	updateErr   error
	refundErr   error
	updated     []domain.Payment
	refunds     []domain.Refund
}

func (r *fakeRefundPaymentRepo) FindByPspId(_ context.Context, _, _ string) (domain.Payment, error) {
	if r.byPspIdErr != nil {
		return domain.Payment{}, r.byPspIdErr
	}
	return r.byPspId, nil
}
func (r *fakeRefundPaymentRepo) Update(_ context.Context, p domain.Payment) (domain.Payment, error) {
	if r.updateErr != nil {
		return domain.Payment{}, r.updateErr
	}
	r.updated = append(r.updated, p)
	return p, nil
}
func (r *fakeRefundPaymentRepo) CreateRefund(_ context.Context, rf domain.Refund) (domain.Refund, error) {
	if r.refundErr != nil {
		return domain.Refund{}, r.refundErr
	}
	r.refunds = append(r.refunds, rf)
	return rf, nil
}

func refundContext() domain.PaymentWebhookContext {
	return domain.PaymentWebhookContext{
		OrgId:   "org_1",
		OrderId: "ord_1",
		Payment: domain.GatewayPayment{PspId: "psp_pay_1", Amount: 5000, Currency: "USD"},
	}
}

func TestPaymentService_ProcessRefund(t *testing.T) {
	t.Run("flips the payment to refunded and records a refund row", func(t *testing.T) {
		repo := &fakeRefundPaymentRepo{byPspId: domain.Payment{OrgId: "org_1", Id: "pmt_1", PspId: "psp_pay_1", Status: domain.PaymentStatusSucceeded}}
		svc := NewPaymentService(repo, silentLogger{})

		got, err := svc.ProcessRefund(context.Background(), refundContext())

		require.NoError(t, err)
		assert.Equal(t, domain.PaymentStatusRefunded, got.Status)
		require.Len(t, repo.updated, 1)
		assert.Equal(t, domain.PaymentStatusRefunded, repo.updated[0].Status)
		require.Len(t, repo.refunds, 1)
		assert.Equal(t, "pmt_1", repo.refunds[0].PaymentId)
		assert.Equal(t, int64(5000), repo.refunds[0].Amount)
	})

	t.Run("payment lookup failure aborts before any write", func(t *testing.T) {
		repo := &fakeRefundPaymentRepo{byPspIdErr: errors.New("not found")}
		svc := NewPaymentService(repo, silentLogger{})

		_, err := svc.ProcessRefund(context.Background(), refundContext())

		require.Error(t, err)
		assert.Empty(t, repo.updated)
		assert.Empty(t, repo.refunds)
	})

	t.Run("update failure aborts before recording the refund", func(t *testing.T) {
		repo := &fakeRefundPaymentRepo{byPspId: domain.Payment{Id: "pmt_1"}, updateErr: errors.New("db down")}
		svc := NewPaymentService(repo, silentLogger{})

		_, err := svc.ProcessRefund(context.Background(), refundContext())

		require.Error(t, err)
		assert.Empty(t, repo.refunds)
	})

	t.Run("refund-row failure is surfaced after the status flip", func(t *testing.T) {
		repo := &fakeRefundPaymentRepo{byPspId: domain.Payment{Id: "pmt_1"}, refundErr: errors.New("db down")}
		svc := NewPaymentService(repo, silentLogger{})

		_, err := svc.ProcessRefund(context.Background(), refundContext())

		require.Error(t, err)
		assert.Len(t, repo.updated, 1, "payment was still updated before the refund row failed")
	})
}
