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

// ----- fakes -----

type hFakeCouponRepo struct {
	port.CouponRepository
	byId    map[string]domain.Coupon
	created []domain.Coupon
}

func newHFakeCouponRepo() *hFakeCouponRepo {
	return &hFakeCouponRepo{byId: map[string]domain.Coupon{}}
}

func (r *hFakeCouponRepo) Create(_ context.Context, c domain.Coupon) (domain.Coupon, error) {
	r.created = append(r.created, c)
	r.byId[c.Id] = c
	return c, nil
}

func (r *hFakeCouponRepo) FindById(_ context.Context, _, id string) (domain.Coupon, error) {
	c, ok := r.byId[id]
	if !ok {
		return domain.Coupon{}, lib.NewCustomError(lib.NotFoundError, "not found", nil)
	}
	return c, nil
}

func (r *hFakeCouponRepo) UpdateMutable(_ context.Context, _, id, name string, active bool, md map[string]string) (domain.Coupon, error) {
	c := r.byId[id]
	c.Name, c.Active, c.Metadata = name, active, md
	r.byId[id] = c
	return c, nil
}

func (r *hFakeCouponRepo) Find(_ context.Context, _ string, _ domain.Pagination) ([]domain.Coupon, int, error) {
	out := make([]domain.Coupon, 0, len(r.byId))
	for _, c := range r.byId {
		out = append(out, c)
	}
	return out, len(out), nil
}

func (r *hFakeCouponRepo) DeleteIfUnreferenced(_ context.Context, _, _ string) error { return nil }

type hFakeCouponCodeRepo struct {
	port.CouponCodeRepository
	byId    map[string]domain.CouponCode
	byCode  map[string]domain.CouponCode
	created []domain.CouponCode
}

func newHFakeCouponCodeRepo() *hFakeCouponCodeRepo {
	return &hFakeCouponCodeRepo{byId: map[string]domain.CouponCode{}, byCode: map[string]domain.CouponCode{}}
}

func (r *hFakeCouponCodeRepo) Create(_ context.Context, c domain.CouponCode) (domain.CouponCode, error) {
	r.created = append(r.created, c)
	r.byId[c.Id] = c
	r.byCode[c.Code] = c
	return c, nil
}

func (r *hFakeCouponCodeRepo) FindByCouponId(_ context.Context, _, couponId string) ([]domain.CouponCode, error) {
	var out []domain.CouponCode
	for _, cc := range r.byId {
		if cc.CouponId == couponId {
			out = append(out, cc)
		}
	}
	return out, nil
}

func (r *hFakeCouponCodeRepo) UpdateMutable(_ context.Context, _, id string, active bool, md map[string]string) (domain.CouponCode, error) {
	c := r.byId[id]
	c.Active, c.Metadata = active, md
	r.byId[id] = c
	return c, nil
}

func (r *hFakeCouponCodeRepo) IncrementRedeemed(_ context.Context, _, id string) error { return nil }

func (r *hFakeCouponCodeRepo) FindByCode(_ context.Context, _, code string) (domain.CouponCode, error) {
	c, ok := r.byCode[code]
	if !ok {
		return domain.CouponCode{}, lib.NewCustomError(lib.NotFoundError, "not found", nil)
	}
	return c, nil
}

type hFakeDiscountRepo struct {
	port.DiscountRepository
	byId map[string]domain.Discount
}

func (r *hFakeDiscountRepo) Create(_ context.Context, d domain.Discount) (domain.Discount, error) {
	if r.byId == nil {
		r.byId = map[string]domain.Discount{}
	}
	r.byId[d.Id] = d
	return d, nil
}
func (r *hFakeDiscountRepo) FindById(_ context.Context, _, id string) (domain.Discount, error) {
	if r.byId != nil {
		if d, ok := r.byId[id]; ok {
			return d, nil
		}
	}
	return domain.Discount{}, lib.NewCustomError(lib.NotFoundError, "not found", nil)
}
func (r *hFakeDiscountRepo) CountByCoupon(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (r *hFakeDiscountRepo) CountByCouponAndCustomer(_ context.Context, _, _, _ string) (int, error) {
	return 0, nil
}

type hFakePriorPayments struct{}

func (h *hFakePriorPayments) HasPriorSuccessfulPayment(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

// newCouponHandlerForTest constructs a CouponHandler backed by in-memory fakes.
func newCouponHandlerForTest(
	t *testing.T,
	cr *hFakeCouponRepo,
	ccr *hFakeCouponCodeRepo,
) *CouponHandler {
	t.Helper()
	svc := service.NewCouponService(cr, ccr, &hFakeDiscountRepo{}, &hFakePriorPayments{}, noopTxManager{}, silentLogger{})
	return NewCouponHandler(svc, silentLogger{}, newRealAuthz(t))
}

// ----- tests -----

func TestCouponHandler_Create(t *testing.T) {
	t.Run("admin creates a coupon and gets it back", func(t *testing.T) {
		cr := newHFakeCouponRepo()
		h := newCouponHandlerForTest(t, cr, newHFakeCouponCodeRepo())

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/coupons", port.CreateCouponInput{
			Name:         "Save 10%",
			DiscountType: "percentage",
			PercentOff:   decimal.NewFromInt(10),
			Duration:     "forever",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got CouponResponse
		decodeJSON(t, rec, &got)
		assert.NotEmpty(t, got.Id)
		assert.Equal(t, "Save 10%", got.Name)
		assert.Equal(t, "percentage", got.DiscountType)
		assert.Equal(t, "forever", got.Duration)
		assert.True(t, got.Active)
		require.Len(t, cr.created, 1)
	})

	t.Run("non-admin (owner) is denied", func(t *testing.T) {
		h := newCouponHandlerForTest(t, newHFakeCouponRepo(), newHFakeCouponCodeRepo())

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/coupons", port.CreateCouponInput{
			Name: "x", DiscountType: "percentage", PercentOff: decimal.NewFromInt(5), Duration: "once",
		})

		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
	})
}

func TestCouponHandler_Get(t *testing.T) {
	t.Run("returns existing coupon", func(t *testing.T) {
		cr := newHFakeCouponRepo()
		h := newCouponHandlerForTest(t, cr, newHFakeCouponCodeRepo())

		// Seed via service to get a real id.
		svc := service.NewCouponService(cr, newHFakeCouponCodeRepo(), &hFakeDiscountRepo{}, &hFakePriorPayments{}, noopTxManager{}, silentLogger{})
		c, err := svc.Create(context.Background(), "org_1", port.CreateCouponInput{
			Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever",
		})
		require.NoError(t, err)

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/coupons/"+c.Id, nil)

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got CouponResponse
		decodeJSON(t, rec, &got)
		assert.Equal(t, c.Id, got.Id)
	})

	t.Run("missing coupon returns not_found", func(t *testing.T) {
		h := newCouponHandlerForTest(t, newHFakeCouponRepo(), newHFakeCouponCodeRepo())

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/coupons/no_such_id", nil)

		assertErrorEnvelope(t, rec, http.StatusNotFound, string(lib.NotFoundError))
	})
}

func TestCouponHandler_List(t *testing.T) {
	cr := newHFakeCouponRepo()
	svc := service.NewCouponService(cr, newHFakeCouponCodeRepo(), &hFakeDiscountRepo{}, &hFakePriorPayments{}, noopTxManager{}, silentLogger{})
	_, _ = svc.Create(context.Background(), "org_1", port.CreateCouponInput{
		Name: "A", DiscountType: "percentage", PercentOff: decimal.NewFromInt(5), Duration: "once",
	})
	_, _ = svc.Create(context.Background(), "org_1", port.CreateCouponInput{
		Name: "B", DiscountType: "percentage", PercentOff: decimal.NewFromInt(10), Duration: "once",
	})

	h := newCouponHandlerForTest(t, cr, newHFakeCouponCodeRepo())
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/coupons?page=0&limit=10", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ListResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 2, got.Meta.Total)
	assert.Equal(t, 10, got.Meta.Limit)
}

func TestCouponHandler_Update(t *testing.T) {
	t.Run("admin can rename and deactivate", func(t *testing.T) {
		cr := newHFakeCouponRepo()
		svc := service.NewCouponService(cr, newHFakeCouponCodeRepo(), &hFakeDiscountRepo{}, &hFakePriorPayments{}, noopTxManager{}, silentLogger{})
		c, _ := svc.Create(context.Background(), "org_1", port.CreateCouponInput{
			Name: "Old", DiscountType: "percentage", PercentOff: decimal.NewFromInt(5), Duration: "once",
		})

		h := newCouponHandlerForTest(t, cr, newHFakeCouponCodeRepo())
		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPatch, "/api/coupons/"+c.Id, port.UpdateCouponInput{
			Name: "New", Active: false,
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got CouponResponse
		decodeJSON(t, rec, &got)
		assert.Equal(t, "New", got.Name)
		assert.False(t, got.Active)
		// discount_type must not change — it is immutable
		assert.Equal(t, "percentage", got.DiscountType)
	})
}

func TestCouponHandler_CreateCode(t *testing.T) {
	t.Run("admin creates a code for a coupon", func(t *testing.T) {
		cr := newHFakeCouponRepo()
		ccr := newHFakeCouponCodeRepo()
		svc := service.NewCouponService(cr, ccr, &hFakeDiscountRepo{}, &hFakePriorPayments{}, noopTxManager{}, silentLogger{})
		c, _ := svc.Create(context.Background(), "org_1", port.CreateCouponInput{
			Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(10), Duration: "forever",
		})

		h := newCouponHandlerForTest(t, cr, ccr)
		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/coupons/"+c.Id+"/codes", port.CreateCouponCodeInput{
			Code: "save10",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got CouponCodeResponse
		decodeJSON(t, rec, &got)
		assert.Equal(t, "SAVE10", got.Code) // service upcases
		assert.Equal(t, c.Id, got.CouponId)
		assert.True(t, got.Active)
	})
}

func TestCouponHandler_GetDiscount(t *testing.T) {
	t.Run("admin gets a discount by id", func(t *testing.T) {
		cr := newHFakeCouponRepo()
		ccr := newHFakeCouponCodeRepo()
		dr := &hFakeDiscountRepo{byId: map[string]domain.Discount{}}

		// Seed a discount directly into the repo.
		d := domain.Discount{
			Id:         "disc_test_1",
			OrgId:      "org_1",
			CouponId:   "cou_abc",
			CustomerId: "cus_xyz",
			OrderId:    "ord_001",
			Status:     domain.DiscountStatusActive,
			StartCycle: 0,
		}
		dr.byId[d.Id] = d

		svc := service.NewCouponService(cr, ccr, dr, &hFakePriorPayments{}, noopTxManager{}, silentLogger{})
		h := NewCouponHandler(svc, silentLogger{}, newRealAuthz(t))

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/discounts/disc_test_1", nil)

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got DiscountResponse
		decodeJSON(t, rec, &got)
		assert.Equal(t, d.Id, got.Id)
		assert.Equal(t, d.CouponId, got.CouponId)
		assert.Equal(t, d.CustomerId, got.CustomerId)
		assert.Equal(t, string(d.Status), got.Status)
	})

	t.Run("missing discount returns not_found", func(t *testing.T) {
		h := newCouponHandlerForTest(t, newHFakeCouponRepo(), newHFakeCouponCodeRepo())

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/discounts/no_such_id", nil)

		assertErrorEnvelope(t, rec, http.StatusNotFound, string(lib.NotFoundError))
	})
}
