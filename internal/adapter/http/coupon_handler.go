package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

type CouponHandler struct {
	service *service.CouponService
	logger  port.Logger
	authz   port.Authz
}

func NewCouponHandler(svc *service.CouponService, logger port.Logger, authz port.Authz) *CouponHandler {
	return &CouponHandler{service: svc, logger: logger, authz: authz}
}

func (h *CouponHandler) RegisterRoutes(s *fuego.Server) {
	g := fuego.Group(s, "/coupons", option.Tags("Coupons"))
	fuego.Post(g, "", h.Create, option.Summary("Create a coupon"))
	fuego.Get(g, "", h.List, option.Summary("List coupons"))
	fuego.Get(g, "/{id}", h.Get, option.Summary("Get a coupon"))
	fuego.Patch(g, "/{id}", h.Update, option.Summary("Update a coupon (name/active/metadata only)"))
	fuego.Delete(g, "/{id}", h.Delete, option.Summary("Delete a coupon"))
	fuego.Post(g, "/{id}/codes", h.CreateCode, option.Summary("Create a redeemable code"))
	fuego.Get(g, "/{id}/codes", h.ListCodes, option.Summary("List a coupon's codes"))

	cg := fuego.Group(s, "/coupon-codes", option.Tags("Coupons"))
	fuego.Patch(cg, "/{id}", h.UpdateCode, option.Summary("Update a coupon code (active/metadata)"))

	dg := fuego.Group(s, "/discounts", option.Tags("Coupons"))
	fuego.Get(dg, "/{id}", h.GetDiscount, option.Summary("Get a discount"))
}

// CouponResponse is the public shape returned for a coupon.
type CouponResponse struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Active       bool   `json:"active"`
	DiscountType string `json:"discount_type"`
	Duration     string `json:"duration"`
}

func couponResponse(c domain.Coupon) CouponResponse {
	return CouponResponse{
		Id:           c.Id,
		Name:         c.Name,
		Active:       c.Active,
		DiscountType: string(c.DiscountType),
		Duration:     string(c.Duration),
	}
}

func (h *CouponHandler) Create(c fuego.ContextWithBody[port.CreateCouponInput]) (CouponResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionCreateCoupon, "") {
		return CouponResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	in, err := c.Body()
	if err != nil {
		return CouponResponse{}, err
	}
	coupon, err := h.service.Create(c.Context(), authUser.OrgId, in)
	if err != nil {
		return CouponResponse{}, NewApiErrorFromError(err)
	}
	return couponResponse(coupon), nil
}

func (h *CouponHandler) Get(c fuego.ContextNoBody) (CouponResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionGetCoupon, c.PathParam("id")) {
		return CouponResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	coupon, err := h.service.Get(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return CouponResponse{}, NewApiErrorFromError(err)
	}
	return couponResponse(coupon), nil
}

func (h *CouponHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionListCoupons, "") {
		return ListResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	pagination := GetPagination(c)
	coupons, total, err := h.service.List(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	out := make([]CouponResponse, len(coupons))
	for i, cp := range coupons {
		out[i] = couponResponse(cp)
	}
	return ListResponse{
		Data: out,
		Meta: Meta{Total: total, Page: pagination.Page, Limit: pagination.Limit},
	}, nil
}

func (h *CouponHandler) Update(c fuego.ContextWithBody[port.UpdateCouponInput]) (CouponResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionUpdateCoupon, c.PathParam("id")) {
		return CouponResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	in, err := c.Body()
	if err != nil {
		return CouponResponse{}, err
	}
	coupon, err := h.service.Update(c.Context(), authUser.OrgId, c.PathParam("id"), in)
	if err != nil {
		return CouponResponse{}, NewApiErrorFromError(err)
	}
	return couponResponse(coupon), nil
}

func (h *CouponHandler) Delete(c fuego.ContextNoBody) (CouponResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionDeleteCoupon, c.PathParam("id")) {
		return CouponResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	if err := h.service.Delete(c.Context(), authUser.OrgId, c.PathParam("id")); err != nil {
		return CouponResponse{}, NewApiErrorFromError(err)
	}
	return CouponResponse{}, nil
}

// CouponCodeResponse is the public shape returned for a coupon code.
type CouponCodeResponse struct {
	Id       string `json:"id"`
	CouponId string `json:"coupon_id"`
	Code     string `json:"code"`
	Active   bool   `json:"active"`
}

func couponCodeResponse(c domain.CouponCode) CouponCodeResponse {
	return CouponCodeResponse{
		Id:       c.Id,
		CouponId: c.CouponId,
		Code:     c.Code,
		Active:   c.Active,
	}
}

func (h *CouponHandler) CreateCode(c fuego.ContextWithBody[port.CreateCouponCodeInput]) (CouponCodeResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionManageCouponCode, c.PathParam("id")) {
		return CouponCodeResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	in, err := c.Body()
	if err != nil {
		return CouponCodeResponse{}, err
	}
	code, err := h.service.CreateCode(c.Context(), authUser.OrgId, c.PathParam("id"), in)
	if err != nil {
		return CouponCodeResponse{}, NewApiErrorFromError(err)
	}
	return couponCodeResponse(code), nil
}

func (h *CouponHandler) ListCodes(c fuego.ContextNoBody) ([]CouponCodeResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionManageCouponCode, c.PathParam("id")) {
		return nil, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	codes, err := h.service.ListCodes(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	out := make([]CouponCodeResponse, len(codes))
	for i, cc := range codes {
		out[i] = couponCodeResponse(cc)
	}
	return out, nil
}

func (h *CouponHandler) UpdateCode(c fuego.ContextWithBody[port.UpdateCouponCodeInput]) (CouponCodeResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionManageCouponCode, c.PathParam("id")) {
		return CouponCodeResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	in, err := c.Body()
	if err != nil {
		return CouponCodeResponse{}, err
	}
	code, err := h.service.UpdateCode(c.Context(), authUser.OrgId, c.PathParam("id"), in)
	if err != nil {
		return CouponCodeResponse{}, NewApiErrorFromError(err)
	}
	return couponCodeResponse(code), nil
}

// DiscountResponse is the public shape returned for a discount.
type DiscountResponse struct {
	Id             string `json:"id"`
	CouponId       string `json:"coupon_id"`
	CustomerId     string `json:"customer_id"`
	SubscriptionId string `json:"subscription_id,omitempty"`
	OrderId        string `json:"order_id,omitempty"`
	Status         string `json:"status"`
	StartCycle     int    `json:"start_cycle"`
}

func discountResponse(d domain.Discount) DiscountResponse {
	return DiscountResponse{
		Id:             d.Id,
		CouponId:       d.CouponId,
		CustomerId:     d.CustomerId,
		SubscriptionId: d.SubscriptionId,
		OrderId:        d.OrderId,
		Status:         string(d.Status),
		StartCycle:     d.StartCycle,
	}
}

func (h *CouponHandler) GetDiscount(c fuego.ContextNoBody) (DiscountResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionGetCoupon, c.PathParam("id")) {
		return DiscountResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	d, err := h.service.GetDiscount(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return DiscountResponse{}, NewApiErrorFromError(err)
	}
	return discountResponse(d), nil
}
