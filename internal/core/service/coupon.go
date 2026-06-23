package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type CouponService struct {
	coupons       port.CouponRepository
	codes         port.CouponCodeRepository
	discounts     port.DiscountRepository
	priorPayments port.PriorPaymentChecker
	tx            port.TxManager
	logger        port.Logger
	reservations  port.CouponReservationRepository
}

func NewCouponService(
	coupons port.CouponRepository,
	codes port.CouponCodeRepository,
	discounts port.DiscountRepository,
	priorPayments port.PriorPaymentChecker,
	tx port.TxManager,
	logger port.Logger,
	reservations port.CouponReservationRepository,
) *CouponService {
	return &CouponService{coupons: coupons, codes: codes, discounts: discounts, priorPayments: priorPayments, tx: tx, logger: logger, reservations: reservations}
}

func (s *CouponService) Create(ctx context.Context, orgId string, in port.CreateCouponInput) (domain.Coupon, error) {
	coupon, err := domain.NewCoupon(domain.NewCouponInput{
		OrgId:             orgId,
		Name:              in.Name,
		DiscountType:      domain.DiscountType(in.DiscountType),
		PercentOff:        in.PercentOff,
		AmountOff:         in.AmountOff,
		Currency:          in.Currency,
		Duration:          domain.Duration(in.Duration),
		DurationInCycles:  in.DurationInCycles,
		RedeemBy:          in.RedeemBy,
		AppliesToProducts: in.AppliesToProducts,
		MaxRedemptions:    in.MaxRedemptions,
		OncePerCustomer:   in.OncePerCustomer,
		Metadata:          in.Metadata,
	})
	if err != nil {
		return domain.Coupon{}, err
	}
	return s.coupons.Create(ctx, coupon)
}

func (s *CouponService) Get(ctx context.Context, orgId, id string) (domain.Coupon, error) {
	return s.coupons.FindById(ctx, orgId, id)
}

func (s *CouponService) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Coupon, int, error) {
	return s.coupons.Find(ctx, orgId, p)
}

func (s *CouponService) Update(ctx context.Context, orgId, id string, in port.UpdateCouponInput) (domain.Coupon, error) {
	return s.coupons.UpdateMutable(ctx, orgId, id, in.Name, in.Active, in.Metadata)
}

func (s *CouponService) Delete(ctx context.Context, orgId, id string) error {
	return s.coupons.DeleteIfUnreferenced(ctx, orgId, id)
}

func (s *CouponService) CreateCode(ctx context.Context, orgId, couponId string, in port.CreateCouponCodeInput) (domain.CouponCode, error) {
	if _, err := s.coupons.FindById(ctx, orgId, couponId); err != nil {
		return domain.CouponCode{}, lib.NewCustomError(lib.NotFoundError, "coupon not found", err)
	}
	code, err := domain.NewCouponCode(domain.NewCouponCodeInput{
		OrgId:          orgId,
		CouponId:       couponId,
		Code:           in.Code,
		CustomerId:     in.CustomerId,
		ExpiresAt:      in.ExpiresAt,
		MaxRedemptions: in.MaxRedemptions,
		Restrictions:   in.Restrictions,
		Metadata:       in.Metadata,
	})
	if err != nil {
		return domain.CouponCode{}, err
	}
	return s.codes.Create(ctx, code)
}

func (s *CouponService) ListCodes(ctx context.Context, orgId, couponId string) ([]domain.CouponCode, error) {
	return s.codes.FindByCouponId(ctx, orgId, couponId)
}

func (s *CouponService) UpdateCode(ctx context.Context, orgId, id string, in port.UpdateCouponCodeInput) (domain.CouponCode, error) {
	return s.codes.UpdateMutable(ctx, orgId, id, in.Active, in.Metadata)
}

type DiscountPreview struct {
	Valid         bool
	Reason        string // set when !Valid (see §5.3)
	DiscountTotal int64
	PerLine       map[string]int64
}

// gateResult carries the resolved coupon/code and any refusal reason.
type gateResult struct {
	coupon  domain.Coupon
	code    domain.CouponCode
	hasCode bool
	reason  string
}

// Validate previews the discount a code would produce. The preview computes the
// discount in isolation and assumes no other discounts are already applied to
// the target; stacking with pre-existing discounts is resolved later at billing time.
func (s *CouponService) Validate(ctx context.Context, orgId, code, customerId, currency string, lines []domain.DiscountableLine) (DiscountPreview, error) {
	gate, err := s.gate(ctx, orgId, code, "", customerId, currency, subtotal(lines))
	if err != nil {
		return DiscountPreview{}, err
	}
	if gate.reason != "" {
		return DiscountPreview{Valid: false, Reason: gate.reason}, nil
	}

	applied := []domain.AppliedDiscount{{
		Coupon:   gate.coupon,
		Discount: domain.Discount{StartCycle: 0, RedeemedAt: time.Now().UTC()},
	}}
	perLine := domain.ApplyDiscounts(lines, applied, 0, currency)
	var total int64
	for _, v := range perLine {
		total += v
	}
	return DiscountPreview{Valid: true, DiscountTotal: total, PerLine: perLine}, nil
}

// gate runs the two-layer §5.3 checks. When code != "" it resolves the code;
// otherwise it resolves the coupon directly (programmatic) by couponId.
func (s *CouponService) gate(ctx context.Context, orgId, code, couponId, customerId, currency string, amount int64) (gateResult, error) {
	var res gateResult

	if code != "" {
		code = strings.ToUpper(strings.TrimSpace(code))
		cc, err := s.codes.FindByCode(ctx, orgId, code)
		if err != nil {
			if errors.Is(err, port.ErrNotFound) {
				return gateResult{reason: "code_not_found"}, nil
			}
			return gateResult{}, err
		}
		res.code = cc
		res.hasCode = true
		couponId = cc.CouponId

		if !cc.Active {
			return gateResult{reason: "inactive"}, nil
		}
		if !cc.ExpiresAt.IsZero() && time.Now().After(cc.ExpiresAt) {
			return gateResult{reason: "code_expired"}, nil
		}
		if cc.MaxRedemptions > 0 {
			held, herr := s.reservations.CountLiveByCode(ctx, orgId, cc.Id, time.Now().UTC())
			if herr != nil {
				return gateResult{}, herr
			}
			if cc.TimesRedeemed+held >= cc.MaxRedemptions {
				return gateResult{reason: "code_cap_reached"}, nil
			}
		}
		if cc.CustomerId != "" && cc.CustomerId != customerId {
			return gateResult{reason: "wrong_customer"}, nil
		}
		if cc.Restrictions.FirstTimeTransaction {
			prior, err := s.priorPayments.HasPriorSuccessfulPayment(ctx, orgId, customerId)
			if err != nil {
				return gateResult{}, err
			}
			if prior {
				return gateResult{reason: "not_first_time"}, nil
			}
		}
		if cc.Restrictions.MinimumAmount > 0 {
			if currency != cc.Restrictions.MinimumAmountCurrency || amount < cc.Restrictions.MinimumAmount {
				return gateResult{reason: "below_minimum"}, nil
			}
		}
	}

	coupon, err := s.coupons.FindById(ctx, orgId, couponId)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return gateResult{reason: "code_not_found"}, nil
		}
		return gateResult{}, err
	}
	res.coupon = coupon

	if !coupon.Active {
		return gateResult{reason: "coupon_inactive"}, nil
	}
	if !coupon.RedeemBy.IsZero() && time.Now().After(coupon.RedeemBy) {
		return gateResult{reason: "expired"}, nil
	}
	if coupon.MaxRedemptions > 0 {
		n, err := s.discounts.CountByCoupon(ctx, orgId, coupon.Id)
		if err != nil {
			return gateResult{}, err
		}
		held, err := s.reservations.CountLiveByCoupon(ctx, orgId, coupon.Id, time.Now().UTC())
		if err != nil {
			return gateResult{}, err
		}
		if n+held >= coupon.MaxRedemptions {
			return gateResult{reason: "cap_reached"}, nil
		}
	}
	if coupon.OncePerCustomer {
		n, err := s.discounts.CountByCouponAndCustomer(ctx, orgId, coupon.Id, customerId)
		if err != nil {
			return gateResult{}, err
		}
		if n == 0 {
			held, herr := s.reservations.ExistsLiveForCustomer(ctx, orgId, coupon.Id, customerId, time.Now().UTC())
			if herr != nil {
				return gateResult{}, herr
			}
			if held {
				n = 1
			}
		}
		if n > 0 {
			return gateResult{reason: "already_used"}, nil
		}
	}
	if coupon.DiscountType == domain.DiscountTypeFixed && coupon.Currency != currency {
		return gateResult{reason: "currency_mismatch"}, nil
	}

	res.reason = ""
	return res, nil
}

func subtotal(lines []domain.DiscountableLine) int64 {
	var t int64
	for _, l := range lines {
		t += l.Total
	}
	return t
}

func (s *CouponService) GetDiscount(ctx context.Context, orgId, id string) (domain.Discount, error) {
	return s.discounts.FindById(ctx, orgId, id)
}

func (s *CouponService) Redeem(ctx context.Context, in port.RedeemCouponInput) (domain.Discount, error) {
	var out domain.Discount
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		gate, err := s.gate(ctx, in.OrgId, in.Code, in.CouponId, in.CustomerId, in.Currency, in.Amount)
		if err != nil {
			return err
		}
		if gate.reason != "" {
			return lib.NewCustomError(lib.BadRequestError, "coupon refused: "+gate.reason, nil)
		}

		couponCodeId := ""
		if gate.hasCode {
			couponCodeId = gate.code.Id
		}
		discount, err := domain.NewDiscount(domain.NewDiscountInput{
			OrgId:          in.OrgId,
			CouponId:       gate.coupon.Id,
			CouponCodeId:   couponCodeId,
			CustomerId:     in.CustomerId,
			SubscriptionId: in.SubscriptionId,
			OrderId:        in.OrderId,
			StartCycle:     in.StartCycle,
		})
		if err != nil {
			return err
		}
		created, err := s.discounts.Create(ctx, discount)
		if err != nil {
			return err
		}
		if gate.hasCode {
			if err := s.codes.IncrementRedeemed(ctx, in.OrgId, gate.code.Id); err != nil {
				return err
			}
		}
		out = created
		return nil
	})
	if err != nil {
		return domain.Discount{}, err
	}
	return out, nil
}

// ReserveInput holds an order's claim on a coupon code's redemption capacity
// while the order is being completed. The hold counts toward caps until it
// expires (lazy) or is consumed/released.
type ReserveInput struct {
	OrgId, Code, CouponId, CustomerId, OrderId, Currency string
	Amount                                               int64
	HoldTTL                                              time.Duration // 0 → default 30m
}

// Reserve atomically gates the coupon and inserts a capacity hold for the order.
// The capacity owner (code → coupon, or coupon directly) is locked FOR UPDATE so
// the cap count + insert can't race another concurrent reserve. A gate refusal
// is returned as a typed CustomError ("coupon refused: <reason>") and rolls the
// tx back, so no hold is written.
func (s *CouponService) Reserve(ctx context.Context, in ReserveInput) (domain.CouponReservation, error) {
	ttl := in.HoldTTL
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	var out domain.CouponReservation
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		// Lock the capacity owner so the cap count + insert is atomic.
		couponId := in.CouponId
		if in.Code != "" {
			cc, err := s.codes.FindByCodeForUpdate(ctx, in.OrgId, in.Code)
			if err != nil {
				if errors.Is(err, port.ErrNotFound) {
					return lib.NewCustomError(lib.ValidationError, "coupon refused: code_not_found", nil)
				}
				return err
			}
			couponId = cc.CouponId
		}
		if _, err := s.coupons.FindByIdForUpdate(ctx, in.OrgId, couponId); err != nil {
			if errors.Is(err, port.ErrNotFound) {
				return lib.NewCustomError(lib.ValidationError, "coupon refused: code_not_found", nil)
			}
			return err
		}

		gate, err := s.gate(ctx, in.OrgId, in.Code, in.CouponId, in.CustomerId, in.Currency, in.Amount)
		if err != nil {
			return err
		}
		if gate.reason != "" {
			return lib.NewCustomError(refusalStatus(gate.reason), "coupon refused: "+gate.reason, nil)
		}

		ccId := ""
		if gate.hasCode {
			ccId = gate.code.Id
		}
		r, err := domain.NewCouponReservation(domain.NewCouponReservationInput{
			OrgId:        in.OrgId,
			CouponId:     gate.coupon.Id,
			CouponCodeId: ccId,
			CustomerId:   in.CustomerId,
			OrderId:      in.OrderId,
			ExpiresAt:    time.Now().UTC().Add(ttl),
		})
		if err != nil {
			return err
		}
		out, err = s.reservations.Create(ctx, r)
		return err
	})
	if err != nil {
		return domain.CouponReservation{}, err
	}
	return out, nil
}

// refusalStatus maps a gate refusal reason to an API error code: capacity/usage
// collisions are conflicts (409), everything else is a validation error (400).
func refusalStatus(reason string) lib.CustomErrorType {
	switch reason {
	case "cap_reached", "code_cap_reached", "already_used":
		return lib.ConflictError
	default:
		return lib.ValidationError
	}
}

// isConflict reports whether err is the storage layer's typed conflict (a
// lib.CustomError of ConflictError type — what asConflictOnUnique produces from
// a Postgres unique violation).
func isConflict(err error) bool {
	var ce lib.CustomError
	return errors.As(err, &ce) && ce.Type == lib.ConflictError
}

// ConsumeInput converts an order's reservation into a Discount on the given
// subscription at payment success.
type ConsumeInput struct {
	OrgId, OrderId, SubscriptionId string
	StartCycle                     int
}

// Consume turns the order's reservation into an active Discount on the
// subscription, increments the code's redemption count, and clears the hold —
// all in one tx. It never re-gates caps (the hold already reserved capacity).
// No-op when the order has no reservation (coupon-less order, or already
// consumed). Idempotent under workflow retry: a unique-violation on the
// (org,coupon,subscription) index means the discount already exists, so the
// stale reservation is just cleared and nil is returned.
func (s *CouponService) Consume(ctx context.Context, in ConsumeInput) (domain.Discount, error) {
	var out domain.Discount
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		rs, err := s.reservations.FindByOrder(ctx, in.OrgId, in.OrderId)
		if err != nil {
			return err
		}
		if len(rs) == 0 {
			return nil // no coupon on this order, or already consumed
		}
		r := rs[0]
		discount, err := domain.NewDiscount(domain.NewDiscountInput{
			OrgId:          in.OrgId,
			CouponId:       r.CouponId,
			CouponCodeId:   r.CouponCodeId,
			CustomerId:     r.CustomerId,
			SubscriptionId: in.SubscriptionId,
			StartCycle:     in.StartCycle,
		})
		if err != nil {
			return err
		}
		created, err := s.discounts.Create(ctx, discount)
		if err != nil {
			if isConflict(err) {
				// Already consumed under a prior retry — just clear the hold.
				return s.reservations.DeleteByOrder(ctx, in.OrgId, in.OrderId)
			}
			return err
		}
		if r.CouponCodeId != "" {
			if err := s.codes.IncrementRedeemed(ctx, in.OrgId, r.CouponCodeId); err != nil {
				return err
			}
		}
		if err := s.reservations.DeleteByOrder(ctx, in.OrgId, in.OrderId); err != nil {
			return err
		}
		out = created
		return nil
	})
	if err != nil {
		return domain.Discount{}, err
	}
	return out, nil
}

// Release drops the order's reservation without converting it (order abandoned
// or failed). Idempotent — deleting a non-existent hold is a no-op.
func (s *CouponService) Release(ctx context.Context, orgId, orderId string) error {
	return s.reservations.DeleteByOrder(ctx, orgId, orderId)
}
