package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payment_links"
	"time"
)

type PaymentLinkController struct {
	paymentLinkService interfaces.PaymentLinkService
	logger             logger.Logger
}

func NewPaymentLinkController(paymentLinkService interfaces.PaymentLinkService, logger logger.Logger) PaymentLinkController {
	return PaymentLinkController{
		paymentLinkService: paymentLinkService,
		logger:             logger,
	}
}

// GetPaymentLink godoc
// @Summary Get a payment link by ID
// @Description Get a payment link by ID
// @Tags payment-links
// @Accept json
// @Produce json
// @Param id path string true "Payment Link ID"
// @Success 200 {object} response.PaymentLinkResponse
// @Failure 400 {object} api.ApiError
// @Failure 404 {object} api.ApiError
// @Failure 500 {object} api.ApiError
// @Router /api/payment-links/{id} [get]
func (c PaymentLinkController) GetPaymentLink(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	id := ctx.Param("id")

	paymentLink, err := c.paymentLinkService.GetPaymentLink(ctx.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert to response DTO
	resp := mapPaymentLinkToResponse(paymentLink)
	ctx.JSON(200, resp)
}

// GetPaymentLinkBySlug godoc
// @Summary Get a payment link by slug
// @Description Get a payment link by slug
// @Tags payment-links
// @Accept json
// @Produce json
// @Param slug path string true "Payment Link Slug"
// @Success 200 {object} response.PaymentLinkResponse
// @Failure 400 {object} api.ApiError
// @Failure 404 {object} api.ApiError
// @Failure 500 {object} api.ApiError
// @Router /api/payment-links/slug/{slug} [get]
func (c PaymentLinkController) GetPaymentLinkBySlug(ctx *gin.Context) {
	slug := ctx.Param("slug")

	paymentLink, err := c.paymentLinkService.GetPaymentLinkBySlug(ctx.Request.Context(), slug)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert to response DTO
	response := mapPaymentLinkToResponse(paymentLink)
	ctx.JSON(200, response)
}

// ListPaymentLinks godoc
// @Summary List payment links
// @Description List payment links with pagination
// @Tags payment-links
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 0)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort_by query string false "Field to sort by (default: created_at)"
// @Param sort_order query string false "Sort order: asc or desc (default: desc)"
// @Success 200 {object} response.PaymentLinkListResponse
// @Failure 400 {object} api.ApiError
// @Failure 500 {object} api.ApiError
// @Router /api/payment-links [get]
func (c PaymentLinkController) ListPaymentLinks(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)

	// Get pagination parameters from request
	pagination := request.GetPagination(ctx)

	// Convert API DTO to application DTO
	appPagination := dto.NewPagination(pagination.Page, pagination.Limit, pagination.SortBy, pagination.SortDirection)

	result, err := c.paymentLinkService.ListPaymentLinks(ctx.Request.Context(), authUser.OrgId, appPagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert to response DTO
	var items []response.PaymentLinkResponse
	for _, paymentLink := range result.Items {
		items = append(items, mapPaymentLinkToResponse(paymentLink))
	}

	ctx.JSON(200, response.PaymentLinkListResponse{
		Items: items,
		Meta: response.Meta{
			Total: result.TotalCount,
			Page: result.Page,
			Limit: result.PageSize,
		},
	})
}

// CreatePaymentLink godoc
// @Summary Create a payment link
// @Description Create a payment link
// @Tags payment-links
// @Accept json
// @Produce json
// @Param payment_link body request.CreatePaymentLinkRequest true "Payment Link"
// @Success 201 {object} response.PaymentLinkResponse
// @Failure 400 {object} api.ApiError
// @Failure 500 {object} api.ApiError
// @Router /api/payment-links [post]
func (c PaymentLinkController) CreatePaymentLink(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)

	var req request.CreatePaymentLinkRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert request to input DTO
	input := payment_links.CreatePaymentLinkInput{
		Slug:      req.Slug,
		Data:      req.Data,
		Config:    req.Config,
		SingleUse: req.SingleUse,
		ExpiresAt: req.ExpiresAt,
	}

	result, err := c.paymentLinkService.CreatePaymentLink(ctx.Request.Context(), authUser.OrgId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert to API response DTO including the token
	baseResponse := mapPaymentLinkToResponse(result.PaymentLink)
	
	// Create a custom response that includes the token
	paymentLinkResponse := struct {
		response.PaymentLinkResponse
		Token string `json:"token"`
	}{
		PaymentLinkResponse: baseResponse,
		Token:               result.Token,
	}
	ctx.JSON(201, paymentLinkResponse)
}

// UpdatePaymentLink godoc
// @Summary Update a payment link
// @Description Update a payment link
// @Tags payment-links
// @Accept json
// @Produce json
// @Param id path string true "Payment Link ID"
// @Param payment_link body request.UpdatePaymentLinkRequest true "Payment Link"
// @Success 200 {object} response.PaymentLinkResponse
// @Failure 400 {object} api.ApiError
// @Failure 404 {object} api.ApiError
// @Failure 500 {object} api.ApiError
// @Router /api/payment-links/{id} [put]
func (c PaymentLinkController) UpdatePaymentLink(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	id := ctx.Param("id")

	var req request.UpdatePaymentLinkRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert request to input DTO
	input := payment_links.UpdatePaymentLinkInput{
		Slug:      req.Slug,
		Data:      req.Data,
		Config:    req.Config,
		SingleUse: req.SingleUse,
		Status:    req.Status,
		ExpiresAt: req.ExpiresAt,
	}

	paymentLink, err := c.paymentLinkService.UpdatePaymentLink(ctx.Request.Context(), authUser.OrgId, id, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert to response DTO
	response := mapPaymentLinkToResponse(paymentLink)
	ctx.JSON(200, response)
}

// DeletePaymentLink godoc
// @Summary Delete a payment link
// @Description Delete a payment link
// @Tags payment-links
// @Accept json
// @Produce json
// @Param id path string true "Payment Link ID"
// @Success 204 "No Content"
// @Failure 400 {object} api.ApiError
// @Failure 404 {object} api.ApiError
// @Failure 500 {object} api.ApiError
// @Router /api/payment-links/{id} [delete]
func (c PaymentLinkController) DeletePaymentLink(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	id := ctx.Param("id")

	err := c.paymentLinkService.DeletePaymentLink(ctx.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.Status(204)
}

// RecordPaymentLinkUsage godoc
// @Summary Record payment link usage
// @Description Record payment link usage
// @Tags payment-links
// @Accept json
// @Produce json
// @Param usage body request.RecordPaymentLinkUsageRequest true "Payment Link Usage"
// @Success 201 {object} response.PaymentLinkUsageResponse
// @Failure 400 {object} api.ApiError
// @Failure 500 {object} api.ApiError
// @Router /api/payment-links/usage [post]
func (c PaymentLinkController) RecordPaymentLinkUsage(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)

	var req request.RecordPaymentLinkUsageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert request to input DTO
	input := payment_links.RecordPaymentLinkUsageInput{
		PaymentLinkId: req.PaymentLinkId,
		SessionId:     req.SessionId,
		CustomerId:    req.CustomerId,
		EventType:     req.EventType,
		IpAddress:     req.IpAddress,
		UserAgent:     req.UserAgent,
		Referer:       req.Referer,
		Country:       req.Country,
		Metadata:      req.Metadata,
	}

	usage, err := c.paymentLinkService.RecordPaymentLinkUsage(ctx.Request.Context(), authUser.OrgId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert to response DTO
	response := mapPaymentLinkUsageToResponse(usage)
	ctx.JSON(201, response)
}

// GetPaymentLinkUsage godoc
// @Summary Get payment link usage by ID
// @Description Get payment link usage by ID
// @Tags payment-links
// @Accept json
// @Produce json
// @Param id path string true "Payment Link Usage ID"
// @Success 200 {object} response.PaymentLinkUsageResponse
// @Failure 400 {object} api.ApiError
// @Failure 404 {object} api.ApiError
// @Failure 500 {object} api.ApiError
// @Router /api/payment-links/usage/{id} [get]
func (c PaymentLinkController) GetPaymentLinkUsage(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	id := ctx.Param("id")

	usage, err := c.paymentLinkService.GetPaymentLinkUsage(ctx.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert to response DTO
	response := mapPaymentLinkUsageToResponse(usage)
	ctx.JSON(200, response)
}

// ListPaymentLinkUsages godoc
// @Summary List payment link usages
// @Description List payment link usages
// @Tags payment-links
// @Accept json
// @Produce json
// @Param id path string true "Payment Link ID"
// @Success 200 {object} response.PaymentLinkUsageListResponse
// @Failure 400 {object} api.ApiError
// @Failure 500 {object} api.ApiError
// @Router /api/payment-links/{id}/usage [get]
func (c PaymentLinkController) ListPaymentLinkUsages(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	paymentLinkId := ctx.Param("id")

	usages, err := c.paymentLinkService.ListPaymentLinkUsages(ctx.Request.Context(), authUser.OrgId, paymentLinkId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert to response DTO
	var items []response.PaymentLinkUsageResponse
	for _, usage := range usages {
		items = append(items, mapPaymentLinkUsageToResponse(usage))
	}

	ctx.JSON(200, response.PaymentLinkUsageListResponse{
		Items: items,
	})
}

// Helper functions to map entities to response DTOs
func mapPaymentLinkToResponse(paymentLink entities.PaymentLink) response.PaymentLinkResponse {
	// Format timestamps
	createdAt := paymentLink.CreatedAt.Format(time.RFC3339)
	updatedAt := paymentLink.UpdatedAt.Format(time.RFC3339)

	var usedAt string
	if !paymentLink.UsedAt.IsZero() {
		usedAt = paymentLink.UsedAt.Format(time.RFC3339)
	}

	var expiresAt string
	if !paymentLink.ExpiresAt.IsZero() {
		expiresAt = paymentLink.ExpiresAt.Format(time.RFC3339)
	}

	return response.PaymentLinkResponse{
		Id:        paymentLink.Id,
		Slug:      paymentLink.Slug,
		Data:      paymentLink.Data,
		Config:    paymentLink.Config,
		SingleUse: paymentLink.SingleUse,
		Status:    paymentLink.Status,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		UsedAt:    usedAt,
		ExpiresAt: expiresAt,
	}
}

func mapPaymentLinkUsageToResponse(usage entities.PaymentLinkUsage) response.PaymentLinkUsageResponse {
	var metadata map[string]interface{}

	// Unmarshal JSON metadata
	if usage.Metadata != nil {
		_ = json.Unmarshal(usage.Metadata, &metadata)
	}

	// Format timestamp
	timestamp := usage.Timestamp.Format(time.RFC3339)

	return response.PaymentLinkUsageResponse{
		Id:            usage.Id,
		PaymentLinkId: usage.PaymentLinkId,
		SessionId:     usage.SessionId,
		CustomerId:    usage.CustomerId,
		EventType:     usage.EventType,
		IpAddress:     usage.IpAddress,
		UserAgent:     usage.UserAgent,
		Referer:       usage.Referer,
		Country:       usage.Country,
		Metadata:      metadata,
		Timestamp:     timestamp,
	}
}
