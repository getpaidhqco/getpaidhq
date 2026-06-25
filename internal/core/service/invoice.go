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
	discounts         port.DiscountRepository
	coupons           port.CouponRepository
}

func NewInvoiceService(
	invoiceRepository port.InvoiceRepository,
	orderRepository port.OrderRepository,
	priceRepository port.PriceRepository,
	usageService *UsageService,
	tx port.TxManager,
	logger port.Logger,
	discounts port.DiscountRepository,
	coupons port.CouponRepository,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepository: invoiceRepository,
		orderRepository:   orderRepository,
		priceRepository:   priceRepository,
		usageService:      usageService,
		tx:                tx,
		logger:            logger,
		discounts:         discounts,
		coupons:           coupons,
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

	// Build the period invoice from the subscription's OWN lines (the recurring
	// order items it bills): a base line for each fixed line, a usage line for each
	// metered line. A subscription owns exactly the lines it should bill, so there
	// is no "primary" arbitration. (ADR 0002.)
	items, err := s.orderRepository.FindOrderItemsBySubscriptionId(ctx, sub.OrgId, sub.Id)
	if err != nil {
		return domain.Invoice{}, err
	}

	inv := domain.NewInvoice(sub, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	productByPrice := map[string]string{}
	for _, it := range items {
		productByPrice[it.PriceId] = it.ProductId
		price, perr := s.priceRepository.FindById(ctx, sub.OrgId, it.PriceId)
		if perr != nil {
			return domain.Invoice{}, perr
		}
		if price.IsMetered() {
			usage, uerr := s.usageService.MeteredUsageForSubscription(ctx, sub, price, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
			if uerr != nil {
				return domain.Invoice{}, uerr
			}
			// A grouped meter splits this charge into one line per discovered group
			// value at the same rate; otherwise it's a single usage line.
			if usage.Grouped != nil {
				for _, g := range usage.Grouped {
					inv.AddLine(domain.UsageLineFromPriceGrouped(sub.OrgId, inv.Id, price, g.Key, g.Value, g.Quantity))
				}
			} else {
				inv.AddLine(domain.UsageLineFromPrice(sub.OrgId, inv.Id, price, usage.Units))
			}
			continue
		}
		// Fixed line → base line. A trial waives the flat fee (ADR 0003).
		if sub.Status != domain.SubscriptionStatusTrial {
			qty := int64(it.Quantity)
			if qty <= 0 {
				qty = 1
			}
			inv.AddLine(domain.BaseLineFromPrice(sub.OrgId, inv.Id, price, decimal.NewFromInt(qty)))
		}
	}

	if err := s.applyDiscounts(ctx, sub, &inv, productByPrice); err != nil {
		return domain.Invoice{}, err
	}

	var created domain.Invoice
	run := func(ctx context.Context) error {
		var e error
		inv.Number, e = s.invoiceRepository.NextInvoiceNumber(ctx, sub.OrgId)
		if e != nil {
			return e
		}
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

// NextInvoiceNumber increments and returns the org-scoped invoice counter.
func (s *InvoiceService) NextInvoiceNumber(ctx context.Context, orgId string) (int64, error) {
	return s.invoiceRepository.NextInvoiceNumber(ctx, orgId)
}

// SetInvoiceCounter sets the org-scoped counter value regardless of the
// current value. The next NextInvoiceNumber call returns value+1.
func (s *InvoiceService) SetInvoiceCounter(ctx context.Context, orgId string, value int64) error {
	return s.invoiceRepository.SetInvoiceCounter(ctx, orgId, value)
}

// applyDiscounts resolves the subscription's active discounts and writes each
// line's DiscountTotal via the pure domain.ApplyDiscounts, scoped to the current
// cycle. No-op when the discount/coupon repos aren't wired (unit tests).
func (s *InvoiceService) applyDiscounts(ctx context.Context, sub domain.Subscription, inv *domain.Invoice, productByPrice map[string]string) error {
	if s.discounts == nil || s.coupons == nil {
		return nil // not wired (unit tests without discounts)
	}
	ds, err := s.discounts.ActiveForSubscription(ctx, sub.OrgId, sub.Id)
	if err != nil {
		return err
	}
	if len(ds) == 0 {
		return nil
	}
	applied := make([]domain.AppliedDiscount, 0, len(ds))
	for _, d := range ds {
		c, err := s.coupons.FindById(ctx, sub.OrgId, d.CouponId)
		if err != nil {
			return err
		}
		applied = append(applied, domain.AppliedDiscount{Discount: d, Coupon: c})
	}
	lines := make([]domain.DiscountableLine, 0, len(inv.LineItems))
	for _, l := range inv.LineItems {
		lines = append(lines, domain.DiscountableLine{LineId: l.Id, ProductId: productByPrice[l.PriceId], Total: l.Total})
	}
	inv.ApplyDiscountTotals(domain.ApplyDiscounts(lines, applied, sub.CyclesProcessed, sub.Currency))
	return nil
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

// MarkOpen finalizes a draft invoice for collection (draft -> open).
func (s *InvoiceService) MarkOpen(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.transition(ctx, orgId, invoiceId, (*domain.Invoice).MarkOpen)
}

// MarkSettled flips an invoice to paid after a succeeded Payment.
func (s *InvoiceService) MarkSettled(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.transition(ctx, orgId, invoiceId, (*domain.Invoice).MarkPaid)
}

// FindCurrentCycle returns the invoice built for a subscription's cycle, or
// port.ErrNotFound if none exists.
func (s *InvoiceService) FindCurrentCycle(ctx context.Context, orgId, subscriptionId string, cycle int) (domain.Invoice, error) {
	return s.invoiceRepository.FindBySubscriptionCycle(ctx, orgId, subscriptionId, cycle)
}

// MarkUncollectible writes off an invoice after recovery is abandoned.
func (s *InvoiceService) MarkUncollectible(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.transition(ctx, orgId, invoiceId, (*domain.Invoice).MarkUncollectible)
}

// Void cancels an invoice that should not be collected.
func (s *InvoiceService) Void(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.transition(ctx, orgId, invoiceId, (*domain.Invoice).Void)
}

func (s *InvoiceService) transition(ctx context.Context, orgId, invoiceId string, apply func(*domain.Invoice) error) (domain.Invoice, error) {
	inv, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		return domain.Invoice{}, err
	}
	if err := apply(&inv); err != nil {
		return domain.Invoice{}, err
	}
	inv.UpdatedAt = time.Now().UTC()
	return s.invoiceRepository.Update(ctx, inv)
}
