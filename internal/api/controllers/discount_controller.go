package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/mappers"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"strconv"
)

type DiscountController struct {
	discountService interfaces.DiscountService
	logger          logger.Logger
}

func NewDiscountController(
	discountService interfaces.DiscountService,
	logger logger.Logger,
) DiscountController {
	return DiscountController{
		discountService: discountService,
		logger:          logger,
	}
}

// GetDiscount retrieves a discount by ID
func (c DiscountController) GetDiscount(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	id := ctx.Param("id")

	discount, err := c.discountService.GetDiscount(ctx.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.JSON(http.StatusOK, mappers.ToDiscountResponse(discount))
}

// ListDiscounts retrieves discounts with pagination
func (c DiscountController) ListDiscounts(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "0"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	sortBy := ctx.DefaultQuery("sort_by", "created_at")
	sortOrder := ctx.DefaultQuery("sort_order", "desc")

	pagination := dto.NewPagination(page, limit, sortBy, sortOrder)

	result, err := c.discountService.ListDiscounts(ctx.Request.Context(), authUser.OrgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.JSON(http.StatusOK, mappers.ToDiscountListResponse(result))
}

// CreateDiscount creates a new discount
func (c DiscountController) CreateDiscount(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)

	var req request.CreateDiscountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	input := mappers.ToCreateDiscountInput(req)
	discount, err := c.discountService.CreateDiscount(ctx.Request.Context(), authUser.OrgId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.JSON(http.StatusCreated, mappers.ToDiscountResponse(discount))
}

// UpdateDiscount updates an existing discount
func (c DiscountController) UpdateDiscount(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	id := ctx.Param("id")

	var req request.UpdateDiscountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	input := mappers.ToUpdateDiscountInput(req)
	discount, err := c.discountService.UpdateDiscount(ctx.Request.Context(), authUser.OrgId, id, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.JSON(http.StatusOK, mappers.ToDiscountResponse(discount))
}

// DeleteDiscount deletes a discount by ID
func (c DiscountController) DeleteDiscount(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	id := ctx.Param("id")

	err := c.discountService.DeleteDiscount(ctx.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.Status(http.StatusNoContent)
}
