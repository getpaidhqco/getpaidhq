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

	// Build one invoice for the period from the subscription's whole order:
	// the flat/base line for this subscription's plan item, plus a usage line for
	// every metered item in the order — all summed. (ADR 0002.)
	items, err := s.orderRepository.FindOrderItemsByOrderId(ctx, sub.OrgId, sub.OrderId)
	if err != nil {
		return domain.Invoice{}, err
	}

	type resolvedItem struct {
		item  domain.OrderItem
		price domain.Price
	}
	resolved := make([]resolvedItem, 0, len(items))
	for _, it := range items {
		p, perr := s.priceRepository.FindById(ctx, sub.OrgId, it.PriceId)
		if perr != nil {
			return domain.Invoice{}, perr
		}
		resolved = append(resolved, resolvedItem{item: it, price: p})
	}

	// The order's metered usage is billed once, by the "primary" subscription —
	// the earliest (lowest order-item id) subscription-category item. This keeps a
	// shared meter from being billed by every plan subscription in the order.
	primaryItemId := ""
	for _, r := range resolved {
		if r.price.Category == domain.PriceCategorySubscription && !r.price.IsMetered() {
			if primaryItemId == "" || r.item.Id < primaryItemId {
				primaryItemId = r.item.Id
			}
		}
	}
	// Defensive: an order with no plan item (only metered) — the calling sub bills it.
	isPrimary := primaryItemId == "" || sub.OrderItemId == primaryItemId

	inv := domain.NewInvoice(sub, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	for _, r := range resolved {
		isOwn := r.item.Id == sub.OrderItemId
		switch {
		case r.price.IsMetered():
			// Metered usage is billed once — by the order's primary subscription, or
			// when it's this subscription's own item. Billed during a trial too (ADR 0003).
			if isOwn || isPrimary {
				units, uerr := s.usageService.UsageForSubscription(ctx, sub, r.price, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
				if uerr != nil {
					return domain.Invoice{}, uerr
				}
				inv.AddLine(domain.UsageLineFromPrice(sub.OrgId, inv.Id, r.price, units))
			}
		case isOwn:
			// This subscription's own (flat/plan) item — the base line. A trial
			// waives the flat fee (ADR 0003).
			if sub.Status != domain.SubscriptionStatusTrial {
				qty := int64(r.item.Quantity)
				if qty <= 0 {
					qty = 1
				}
				inv.AddLine(domain.BaseLineFromPrice(sub.OrgId, inv.Id, r.price, decimal.NewFromInt(qty)))
			}
		}
		// Sibling non-metered items belong to other subscriptions — skipped.
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

// GetById returns one invoice (with line items).
func (s *InvoiceService) GetById(ctx context.Context, orgId, id string) (domain.Invoice, error) {
	return s.invoiceRepository.FindById(ctx, orgId, id)
}

// List returns the org's invoices, newest first.
func (s *InvoiceService) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Invoice, int, error) {
	return s.invoiceRepository.List(ctx, orgId, p)
}

// ListBySubscription returns a subscription's invoices.
func (s *InvoiceService) ListBySubscription(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.Invoice, int, error) {
	return s.invoiceRepository.FindBySubscriptionId(ctx, orgId, subscriptionId, p)
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
