package service

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// InvoiceService builds and persists the per-cycle invoice and flips its status
// after settlement. Narrow — no workflow engine, no signaling. It resolves the
// subscription's Price via the order-item + price repos (the domain is ID-only —
// nothing is embedded on the Subscription aggregate).
type InvoiceService struct {
	invoiceRepository port.InvoiceRepository
	orderRepository   port.OrderRepository
	priceRepository   port.PriceRepository
	usageService      *UsageService
	tx                port.TxManager
	logger            port.Logger
}

func NewInvoiceService(
	invoiceRepository port.InvoiceRepository,
	orderRepository port.OrderRepository,
	priceRepository port.PriceRepository,
	usageService *UsageService,
	tx port.TxManager,
	logger port.Logger,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepository: invoiceRepository,
		orderRepository:   orderRepository,
		priceRepository:   priceRepository,
		usageService:      usageService,
		tx:                tx,
		logger:            logger,
	}
}

// BuildForBillingPeriod builds (or returns the already-built) invoice for the
// subscription's current cycle and persists it as draft. Idempotent on
// (orgId, subscriptionId, cycle) so a replayed billing run reuses one invoice.
func (s *InvoiceService) BuildForBillingPeriod(ctx context.Context, sub domain.Subscription) (domain.Invoice, error) {
	existing, err := s.invoiceRepository.FindBySubscriptionCycle(ctx, sub.OrgId, sub.Id, sub.CyclesProcessed)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, port.ErrNotFound) {
		return domain.Invoice{}, err
	}

	// Resolve the linked Price via the order item (domain is ID-only).
	item, err := s.orderRepository.FindOrderItemById(ctx, sub.OrgId, sub.OrderItemId)
	if err != nil {
		return domain.Invoice{}, err
	}
	price, err := s.priceRepository.FindById(ctx, sub.OrgId, item.PriceId)
	if err != nil {
		return domain.Invoice{}, err
	}

	var inv domain.Invoice
	if price.Category == domain.PriceCategoryMetered {
		// Metered price: aggregate the cycle's usage into a single usage line.
		// No base line — the charge is the measured units priced by the scheme.
		units, uerr := s.usageService.UsageForSubscription(ctx, sub, price, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
		if uerr != nil {
			return domain.Invoice{}, uerr
		}
		inv = domain.NewInvoice(sub, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
		inv.AddLine(domain.UsageLineFromPrice(sub.OrgId, inv.Id, price, units))
	} else {
		qty := int64(item.Quantity)
		if qty <= 0 {
			qty = 1
		}
		inv = domain.BuildInvoiceForPeriod(sub, price, decimal.NewFromInt(qty), sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	}

	var created domain.Invoice
	run := func(ctx context.Context) error {
		var e error
		created, e = s.invoiceRepository.Create(ctx, inv)
		return e
	}
	if s.tx != nil {
		err = s.tx.RunInTx(ctx, run)
	} else {
		err = run(ctx) // tests without a TxManager
	}
	if err != nil {
		return domain.Invoice{}, err
	}
	s.logger.Infof("[%s][%s] invoice %s built for cycle %d total=%d", sub.OrgId, sub.Id, created.Id, created.Cycle, created.Total)
	return created, nil
}

// MarkSettled flips an invoice to paid after a succeeded Payment.
func (s *InvoiceService) MarkSettled(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.setStatus(ctx, orgId, invoiceId, domain.InvoiceStatusPaid)
}

// MarkUnpaid flips an invoice to unpaid after a failed/exhausted settlement.
func (s *InvoiceService) MarkUnpaid(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.setStatus(ctx, orgId, invoiceId, domain.InvoiceStatusUnpaid)
}

func (s *InvoiceService) setStatus(ctx context.Context, orgId, invoiceId string, status domain.InvoiceStatus) (domain.Invoice, error) {
	inv, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		return domain.Invoice{}, err
	}
	inv.Status = status
	inv.UpdatedAt = time.Now().UTC()
	return s.invoiceRepository.Update(ctx, inv)
}
