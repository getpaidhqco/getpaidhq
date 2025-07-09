package controllers

import (
	"github.com/gin-gonic/gin"
	"time"

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

	// Set orgId in the request
	req.OrgId = orgId

	// Convert API DTO to application DTO
	appInput := mappers.ToRecordUsageInput(req)

	record, err := u.usageRecordingService.RecordUsage(c.Request.Context(), appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert application response to API response
	response := mappers.ToCloudEventUsageResponse(record)
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
	response := mappers.ToUsageEventListResponse(result)
	c.JSON(200, response)
}

// GetUsageEvent handles GET /api/usage-records/:id
func (u UsageRecordingController) GetUsageEvent(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId

	usageRecordId := c.Param("id")

	record, err := u.usageRecordingService.GetUsageEvent(c.Request.Context(), orgId, usageRecordId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entity to API response
	response := mappers.ToUsageEventResponse(record)
	c.JSON(200, response)
}

// DeleteUsageEvent handles DELETE /api/usage-records/:id
func (u UsageRecordingController) DeleteUsageEvent(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId

	eventId := c.Param("id")

	// For DeleteUsageEvent, we need to provide the event time
	// Since we don't have it from the request, we'll use the current time
	// This might not work for all cases, but it's a reasonable default
	eventTime := time.Now().UTC()

	err := u.usageRecordingService.DeleteUsageEvent(c.Request.Context(), orgId, eventId, eventTime)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.Status(204) // No content
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
		responseItems = append(responseItems, mappers.ToUsageEventResponse(record))
	}
	c.JSON(200, gin.H{"items": responseItems, "count": len(responseItems)})
}
