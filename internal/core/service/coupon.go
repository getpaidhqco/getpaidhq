package service

import (
	"context"
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
}

func NewCouponService(
	coupons port.CouponRepository,
	codes port.CouponCodeRepository,
	discounts port.DiscountRepository,
	priorPayments port.PriorPaymentChecker,
	tx port.TxManager,
	logger port.Logger,
) *CouponService {
	return &CouponService{coupons: coupons, codes: codes, discounts: discounts, priorPayments: priorPayments, tx: tx, logger: logger}
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
			return gateResult{reason: "code_not_found"}, nil
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
		if cc.MaxRedemptions > 0 && cc.TimesRedeemed >= cc.MaxRedemptions {
			return gateResult{reason: "code_cap_reached"}, nil
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
		return gateResult{reason: "code_not_found"}, nil
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
		if n >= coupon.MaxRedemptions {
			return gateResult{reason: "cap_reached"}, nil
		}
	}
	if coupon.OncePerCustomer {
		n, err := s.discounts.CountByCouponAndCustomer(ctx, orgId, coupon.Id, customerId)
		if err != nil {
			return gateResult{}, err
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
