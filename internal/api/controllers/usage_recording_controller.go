package controllers

import (
	"time"

	"github.com/gin-gonic/gin"

	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/mappers"
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

	// Convert API DTO to application DTO
	appInput := mappers.ToRecordUsageInput(req)

	record, err := u.usageRecordingService.RecordUsage(c.Request.Context(), orgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entity to API response
	response := mappers.ToUsageRecordResponse(record)
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

	// Convert API DTO to application DTO
	appInput := mappers.ToBatchRecordUsageInput(req)

	records, err := u.usageRecordingService.BatchRecordUsage(c.Request.Context(), orgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entities to API response
	var responseItems []interface{}
	for _, record := range records {
		responseItems = append(responseItems, mappers.ToUsageRecordResponse(record))
	}
	c.JSON(201, gin.H{"items": responseItems, "count": len(responseItems)})
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
	pagination := request.GetPagination(c)

	// Convert API DTO to application DTO
	appInput := mappers.ToListUsageRecordsInput(subscriptionItemId, pagination)

	result, err := u.usageRecordingService.ListUsageRecords(c.Request.Context(), orgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert paginated result to API response
	response := mappers.ToUsageRecordListResponse(result)
	c.JSON(200, response)
}

// GetUsageRecord handles GET /api/usage-records/:id
func (u UsageRecordingController) GetUsageRecord(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId

	usageRecordId := c.Param("id")

	record, err := u.usageRecordingService.GetUsageRecord(c.Request.Context(), orgId, usageRecordId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entity to API response
	response := mappers.ToUsageRecordResponse(record)
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

	// Create request DTO and convert to application DTO
	req := request.GetUsageSummaryRequest{
		SubscriptionItemId: subscriptionItemId,
		StartDate:          startDate,
		EndDate:            endDate,
	}
	appInput := mappers.ToUsageSummaryInput(req)

	result, err := u.usageRecordingService.GetUsageSummary(c.Request.Context(), orgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert application result to API response
	response := mappers.ToUsageSummaryResponse(result)
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

	// Convert parameters to application DTO
	appInput := mappers.ToGetSubscriptionUsageInput(subscriptionId, startDateStr, endDateStr)

	records, err := u.usageRecordingService.GetSubscriptionUsage(c.Request.Context(), orgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entities to API response
	var responseItems []interface{}
	for _, record := range records {
		responseItems = append(responseItems, mappers.ToUsageRecordResponse(record))
	}
	c.JSON(200, gin.H{"items": responseItems, "count": len(responseItems)})
}

// DeleteUsageRecord handles DELETE /api/usage-records/:id
func (u UsageRecordingController) DeleteUsageRecord(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId

	usageRecordId := c.Param("id")

	err := u.usageRecordingService.DeleteUsageRecord(c.Request.Context(), orgId, usageRecordId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(204, nil)
}
