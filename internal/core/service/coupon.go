package service

import (
	"context"

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

// Validate and Redeem are added in the next tasks.
