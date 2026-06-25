package service

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// MeteredUsageReader is the narrow read InvoiceService needs from usage: a
// metered price's usage for a billing window. Defined here (not in port/) because
// the return type MeteredUsage lives in this package — a port-level interface would
// import-cycle. *UsageService satisfies it (assertion below).
type MeteredUsageReader interface {
	MeteredUsageForSubscription(ctx context.Context, sub domain.Subscription, price domain.Price, from, to time.Time) (MeteredUsage, error)
}

var _ MeteredUsageReader = (*UsageService)(nil)

// InvoiceService builds and persists the per-cycle invoice and flips its status
// after settlement. Narrow — no workflow engine, no signaling. It resolves the
// subscription's Price via the order-item + price repos (the domain is ID-only —
// nothing is embedded on the Subscription aggregate).
type InvoiceService struct {
	invoiceRepository           port.InvoiceRepository
	orderRepository             port.OrderRepository
	priceRepository             port.PriceRepository
	subscriptionRepository      port.SubscriptionRepository
	usageService                MeteredUsageReader
	tx                          port.TxManager
	logger                      port.Logger
	discountRepository          port.DiscountRepository
	couponRepository            port.CouponRepository
	couponReservationRepository port.CouponReservationRepository
	invoiceSettingsResolver     port.InvoiceSettingsResolver
}

func NewInvoiceService(
	invoiceRepository port.InvoiceRepository,
	orderRepository port.OrderRepository,
	priceRepository port.PriceRepository,
	subscriptionRepository port.SubscriptionRepository,
	usageService MeteredUsageReader,
	tx port.TxManager,
	logger port.Logger,
	discountRepository port.DiscountRepository,
	couponRepository port.CouponRepository,
	couponReservationRepository port.CouponReservationRepository,
	invoiceSettingsResolver port.InvoiceSettingsResolver,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepository:           invoiceRepository,
		orderRepository:             orderRepository,
		priceRepository:             priceRepository,
		subscriptionRepository:      subscriptionRepository,
		usageService:                usageService,
		tx:                          tx,
		logger:                      logger,
		discountRepository:          discountRepository,
		couponRepository:            couponRepository,
		couponReservationRepository: couponReservationRepository,
		invoiceSettingsResolver:     invoiceSettingsResolver,
	}
}

// reference resolves the org's invoice settings and formats the human reference.
func (s *InvoiceService) reference(ctx context.Context, orgId string, number int64) string {
	cfg, err := s.invoiceSettingsResolver.ResolveInvoiceSettings(ctx, orgId)
	if err != nil {
		cfg = domain.DefaultInvoiceSettings()
	}
	return cfg.FormatReference(number)
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
		inv.Reference = s.reference(ctx, sub.OrgId, inv.Number)
		created, e = s.invoiceRepository.Create(ctx, inv)
		return e
	}
	err = s.tx.RunInTx(ctx, run)
	if err != nil {
		return domain.Invoice{}, err
	}
	s.logger.Infof("[%s][%s] invoice %s built for cycle %d total=%d", sub.OrgId, sub.Id, created.Id, created.Cycle, created.Total)
	return created, nil
}

// BuildForOrder builds (or returns) the order's combined cycle-0 invoice: each
// subscription's first-period line(s) + every one-time line, with the order's
// discount applied. Idempotent on the order. Status is left as the domain
// default (the caller marks open/settled). Number + Reference set in the tx.
func (s *InvoiceService) BuildForOrder(ctx context.Context, order domain.Order) (domain.Invoice, error) {
	existing, err := s.invoiceRepository.FindOrderInvoice(ctx, order.OrgId, order.Id)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, port.ErrNotFound) {
		return domain.Invoice{}, err
	}

	items, err := s.orderRepository.FindOrderItemsByOrderId(ctx, order.OrgId, order.Id)
	if err != nil {
		return domain.Invoice{}, err
	}
	if len(items) == 0 {
		return domain.Invoice{}, port.ErrNotFound // nothing to invoice
	}

	// Subscriptions on the order give us the first-period dates and the cycle-0
	// linkage. With exactly one subscription this invoice IS that subscription's
	// cycle-0 invoice (SubscriptionId set + its CurrentPeriod*), so the billing
	// engine won't rebuild cycle 0. A pure one-time order has no subscription.
	var sub domain.Subscription
	var hasSub bool
	subs, serr := s.subscriptionRepository.FindByOrderId(ctx, order.OrgId, order.Id)
	if serr != nil {
		return domain.Invoice{}, serr
	}
	if len(subs) == 1 {
		sub = subs[0]
		hasSub = true
	}

	now := time.Now().UTC()
	periodStart, periodEnd := order.UpdatedAt, order.UpdatedAt
	if periodStart.IsZero() {
		periodStart, periodEnd = now, now
	}
	if hasSub {
		periodStart, periodEnd = sub.CurrentPeriodStart, sub.CurrentPeriodEnd
	}

	inv := domain.Invoice{
		OrgId:       order.OrgId,
		Id:          lib.GenerateId("inv"),
		OrderId:     order.Id,
		CustomerId:  order.CustomerId,
		Status:      domain.InvoiceStatusDraft,
		Currency:    order.Currency,
		Cycle:       0,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if hasSub {
		inv.SubscriptionId = sub.Id
	}

	// Build a line per order item: recurring fixed → base line over the first
	// period; recurring metered → usage line (cycle 0 has ~no usage yet);
	// one-time → base line. Shared with BuildForBillingPeriod via addItemLine.
	productByPrice := map[string]string{}
	for _, it := range items {
		productByPrice[it.PriceId] = it.ProductId
		price, perr := s.priceRepository.FindById(ctx, order.OrgId, it.PriceId)
		if perr != nil {
			return domain.Invoice{}, perr
		}
		if err := s.addItemLine(ctx, &inv, price, it, hasSub, sub, periodStart, periodEnd); err != nil {
			return domain.Invoice{}, err
		}
	}

	if err := s.applyOrderDiscounts(ctx, order, &inv, productByPrice); err != nil {
		return domain.Invoice{}, err
	}

	var created domain.Invoice
	run := func(ctx context.Context) error {
		var e error
		inv.Number, e = s.invoiceRepository.NextInvoiceNumber(ctx, order.OrgId)
		if e != nil {
			return e
		}
		inv.Reference = s.reference(ctx, order.OrgId, inv.Number)
		created, e = s.invoiceRepository.Create(ctx, inv)
		return e
	}
	err = s.tx.RunInTx(ctx, run)
	if err != nil {
		return domain.Invoice{}, err
	}
	s.logger.Infof("[%s] order invoice %s built for order %s total=%d", order.OrgId, created.Id, order.Id, created.Total)
	return created, nil
}

// addItemLine appends the invoice line(s) for one order item: a usage line (or
// grouped usage lines) for a metered price, otherwise a base line. When no
// subscription is present (pure one-time order), a metered line falls back to a
// zero-units usage line — there is nothing to measure usage against yet.
func (s *InvoiceService) addItemLine(ctx context.Context, inv *domain.Invoice, price domain.Price, it domain.OrderItem, hasSub bool, sub domain.Subscription, periodStart, periodEnd time.Time) error {
	if price.IsMetered() {
		var units decimal.Decimal
		if hasSub {
			usage, uerr := s.usageService.MeteredUsageForSubscription(ctx, sub, price, periodStart, periodEnd)
			if uerr != nil {
				return uerr
			}
			if usage.Grouped != nil {
				for _, g := range usage.Grouped {
					inv.AddLine(domain.UsageLineFromPriceGrouped(inv.OrgId, inv.Id, price, g.Key, g.Value, g.Quantity))
				}
				return nil
			}
			units = usage.Units
		}
		inv.AddLine(domain.UsageLineFromPrice(inv.OrgId, inv.Id, price, units))
		return nil
	}
	qty := int64(it.Quantity)
	if qty <= 0 {
		qty = 1
	}
	inv.AddLine(domain.BaseLineFromPrice(inv.OrgId, inv.Id, price, decimal.NewFromInt(qty)))
	return nil
}

// applyOrderDiscounts resolves the order's discount and writes each line's
// DiscountTotal via the pure domain.ApplyDiscounts at cycle 0. It resolves
// committed-OR-reservation:
//
//   - if the order already has a committed (order-owned) Discount, that is used
//     (the post-completion / billing path);
//   - otherwise, if the order has a live coupon reservation (the pre-payment
//     upfront-invoice path, before the reservation is consumed into a Discount),
//     a synthetic AppliedDiscount is built from the reserved coupon so the open
//     invoice already carries the discounted total. The synthetic Discount is
//     NOT persisted — it is only used to compute the bill. At completion, Consume
//     creates the real Discount and BuildForOrder returns the existing invoice,
//     so totals stay consistent.
//
// No-op when the order has neither a committed discount nor a reservation.
func (s *InvoiceService) applyOrderDiscounts(ctx context.Context, order domain.Order, inv *domain.Invoice, productByPrice map[string]string) error {
	ds, err := s.discountRepository.ActiveForOrder(ctx, order.OrgId, order.Id)
	if err != nil {
		return err
	}

	applied := make([]domain.AppliedDiscount, 0, len(ds))
	if len(ds) > 0 {
		// Committed discounts (post-completion / billing path).
		for _, d := range ds {
			c, err := s.couponRepository.FindById(ctx, order.OrgId, d.CouponId)
			if err != nil {
				return err
			}
			applied = append(applied, domain.AppliedDiscount{Discount: d, Coupon: c})
		}
	} else {
		// No committed discount yet — fall back to the order's live reservation
		// so a pre-payment (open) invoice is still discounted.
		rs, rerr := s.couponReservationRepository.FindByOrder(ctx, order.OrgId, order.Id)
		if rerr != nil {
			return rerr
		}
		if len(rs) == 0 {
			return nil
		}
		r := rs[0]
		c, cerr := s.couponRepository.FindById(ctx, order.OrgId, r.CouponId)
		if cerr != nil {
			return cerr
		}
		applied = append(applied, domain.AppliedDiscount{
			Coupon: c,
			Discount: domain.Discount{
				OrgId:      order.OrgId,
				CouponId:   r.CouponId,
				CustomerId: r.CustomerId,
				OrderId:    order.Id,
				StartCycle: 0,
				Status:     domain.DiscountStatusActive,
				RedeemedAt: time.Now().UTC(),
			},
		})
	}

	if len(applied) == 0 {
		return nil
	}
	lines := make([]domain.DiscountableLine, 0, len(inv.LineItems))
	for _, l := range inv.LineItems {
		lines = append(lines, domain.DiscountableLine{LineId: l.Id, ProductId: productByPrice[l.PriceId], Total: l.Total})
	}
	inv.ApplyDiscountTotals(domain.ApplyDiscounts(lines, applied, 0, order.Currency))
	return nil
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
// cycle. No-op when the subscription has no active discounts.
func (s *InvoiceService) applyDiscounts(ctx context.Context, sub domain.Subscription, inv *domain.Invoice, productByPrice map[string]string) error {
	ds, err := s.discountRepository.ActiveForSubscription(ctx, sub.OrgId, sub.Id)
	if err != nil {
		return err
	}
	if len(ds) == 0 {
		return nil
	}
	applied := make([]domain.AppliedDiscount, 0, len(ds))
	for _, d := range ds {
		c, err := s.couponRepository.FindById(ctx, sub.OrgId, d.CouponId)
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
