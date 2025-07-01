package controllers

import (
    "strconv"
    "time"

    "github.com/gin-gonic/gin"

    "payloop/internal/api"
    "payloop/internal/api/authn"
    "payloop/internal/api/dto/request"
    "payloop/internal/application/interfaces"
    "payloop/internal/application/lib/logger"
    "payloop/internal/lib"
)

type UsageRecordingController struct {
    usageRecordingService interfaces.UsageRecordingService
    logger                logger.Logger
}

func NewUsageRecordingController(
    usageRecordingService interfaces.UsageRecordingService,
    logger logger.Logger,
) UsageRecordingController {
    return UsageRecordingController{
        usageRecordingService: usageRecordingService,
        logger:                logger,
    }
}

// RecordUsage handles POST /api/usage-records
func (u UsageRecordingController) RecordUsage(c *gin.Context) {
    user, _ := c.Get("user")
    authUser := user.(authn.User)
    orgId := authUser.OrgId

    var req request.RecordUsageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        apiErr := api.NewApiError(lib.BadRequestError, "Invalid request body", err.Error())
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    response, err := u.usageRecordingService.RecordUsage(c.Request.Context(), orgId, req)
    if err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    c.JSON(201, response)
}

// BatchRecordUsage handles POST /api/usage-records/batch  
func (u UsageRecordingController) BatchRecordUsage(c *gin.Context) {
    user, _ := c.Get("user")
    authUser := user.(authn.User)
    orgId := authUser.OrgId

    var req request.BatchRecordUsageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        apiErr := api.NewApiError(lib.BadRequestError, "Invalid request body", err.Error())
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    response, err := u.usageRecordingService.BatchRecordUsage(c.Request.Context(), orgId, req)
    if err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    c.JSON(201, response)
}

// ListUsageRecords handles GET /api/usage-records
func (u UsageRecordingController) ListUsageRecords(c *gin.Context) {
    user, _ := c.Get("user")
    authUser := user.(authn.User)
    orgId := authUser.OrgId

    subscriptionItemId := c.Query("subscription_item_id")
    if subscriptionItemId == "" {
        apiErr := api.NewApiError(lib.BadRequestError, "subscription_item_id is required", nil)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    // Parse pagination parameters
    limit := 50
    if limitStr := c.Query("limit"); limitStr != "" {
        if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
            limit = l
        }
    }

    offset := 0
    if offsetStr := c.Query("offset"); offsetStr != "" {
        if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
            offset = o
        }
    }

    response, err := u.usageRecordingService.ListUsageRecords(
        c.Request.Context(), orgId, subscriptionItemId, limit, offset)
    if err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    c.JSON(200, response)
}

// GetUsageRecord handles GET /api/usage-records/:id
func (u UsageRecordingController) GetUsageRecord(c *gin.Context) {
    user, _ := c.Get("user")
    authUser := user.(authn.User)
    orgId := authUser.OrgId

    usageRecordId := c.Param("id")

    response, err := u.usageRecordingService.GetUsageRecord(
        c.Request.Context(), orgId, usageRecordId)
    if err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    c.JSON(200, response)
}

// GetUsageSummary handles GET /api/subscription-items/:id/usage-summary
func (u UsageRecordingController) GetUsageSummary(c *gin.Context) {
    user, _ := c.Get("user")
    authUser := user.(authn.User)
    orgId := authUser.OrgId

    subscriptionItemId := c.Param("id")

    // Parse query parameters
    startDateStr := c.Query("start_date")
    endDateStr := c.Query("end_date")
    // granularity can be used in the future if needed
    _ = c.DefaultQuery("granularity", "day")

    var startDate, endDate time.Time
    var err error

    if startDateStr != "" {
        startDate, err = time.Parse(time.RFC3339, startDateStr)
        if err != nil {
            apiErr := api.NewApiError(lib.BadRequestError, "Invalid start_date format", err.Error())
            c.JSON(apiErr.GetHttpErrorCode(), apiErr)
            return
        }
    }

    if endDateStr != "" {
        endDate, err = time.Parse(time.RFC3339, endDateStr)
        if err != nil {
            apiErr := api.NewApiError(lib.BadRequestError, "Invalid end_date format", err.Error())
            c.JSON(apiErr.GetHttpErrorCode(), apiErr)
            return
        }
    }

    req := request.GetUsageSummaryRequest{
        SubscriptionItemId: subscriptionItemId,
        StartDate:          startDate,
        EndDate:            endDate,
    }

    response, err := u.usageRecordingService.GetUsageSummary(c.Request.Context(), orgId, req)
    if err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    c.JSON(200, response)
}

// GetSubscriptionUsage handles GET /api/subscriptions/:id/usage
func (u UsageRecordingController) GetSubscriptionUsage(c *gin.Context) {
    user, _ := c.Get("user")
    authUser := user.(authn.User)
    orgId := authUser.OrgId

    subscriptionId := c.Param("id")

    // Parse query parameters for date range
    startDateStr := c.Query("start_date")
    endDateStr := c.Query("end_date")

    var startDate, endDate time.Time
    var err error

    if startDateStr != "" {
        startDate, err = time.Parse(time.RFC3339, startDateStr)
        if err != nil {
            apiErr := api.NewApiError(lib.BadRequestError, "Invalid start_date format", err.Error())
            c.JSON(apiErr.GetHttpErrorCode(), apiErr)
            return
        }
    }

    if endDateStr != "" {
        endDate, err = time.Parse(time.RFC3339, endDateStr)
        if err != nil {
            apiErr := api.NewApiError(lib.BadRequestError, "Invalid end_date format", err.Error())
            c.JSON(apiErr.GetHttpErrorCode(), apiErr)
            return
        }
    }

    response, err := u.usageRecordingService.GetSubscriptionUsage(
        c.Request.Context(), orgId, subscriptionId, startDate, endDate)
    if err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    c.JSON(200, response)
}

// DeleteUsageRecord handles DELETE /api/usage-records/:id
func (u UsageRecordingController) DeleteUsageRecord(c *gin.Context) {
    user, _ := c.Get("user")
    authUser := user.(authn.User)
    orgId := authUser.OrgId

    usageRecordId := c.Param("id")

    err := u.usageRecordingService.DeleteUsageRecord(
        c.Request.Context(), orgId, usageRecordId)
    if err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    c.JSON(204, nil)
}
