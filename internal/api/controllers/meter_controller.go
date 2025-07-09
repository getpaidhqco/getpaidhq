package controllers

import (
	"github.com/gin-gonic/gin"

	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/mappers"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type MeterController struct {
	meterService interfaces.MeterService
	logger       logger.Logger
}

func NewMeterController(
	meterService interfaces.MeterService,
	logger logger.Logger,
) MeterController {
	return MeterController{
		meterService: meterService,
		logger:       logger,
	}
}

// Create handles POST /api/meters
func (m MeterController) Create(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId

	var req request.CreateMeterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiError(lib.BadRequestError, "Invalid request body", err.Error())
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert API DTO to application DTO
	appInput := mappers.ToCreateMeterInput(req)

	meter, err := m.meterService.Create(c.Request.Context(), orgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entity to API response
	response := mappers.ToMeterResponse(meter)
	c.JSON(201, response)
}

// Update handles PUT /api/meters/:id
func (m MeterController) Update(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	meterId := c.Param("id")

	var req request.UpdateMeterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiError(lib.BadRequestError, "Invalid request body", err.Error())
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert API DTO to application DTO
	appInput := mappers.ToUpdateMeterInput(req)

	meter, err := m.meterService.Update(c.Request.Context(), orgId, meterId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entity to API response
	response := mappers.ToMeterResponse(meter)
	c.JSON(200, response)
}

// Get handles GET /api/meters/:id
func (m MeterController) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	meterId := c.Param("id")

	meter, err := m.meterService.Get(c.Request.Context(), orgId, meterId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entity to API response
	response := mappers.ToMeterResponse(meter)
	c.JSON(200, response)
}

// GetBySlug handles GET /api/meters/slug/:slug
func (m MeterController) GetBySlug(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	slug := c.Param("slug")

	meter, err := m.meterService.GetBySlug(c.Request.Context(), orgId, slug)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert domain entity to API response
	response := mappers.ToMeterResponse(meter)
	c.JSON(200, response)
}

// List handles GET /api/meters
func (m MeterController) List(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId

	// Parse pagination parameters
	pagination := request.GetPagination(c)

	result, err := m.meterService.List(c.Request.Context(), orgId, mappers.ToPagination(pagination))
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert paginated result to API response
	response := mappers.ToMeterListResponse(result)
	c.JSON(200, response)
}

// Delete handles DELETE /api/meters/:id
func (m MeterController) Delete(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	meterId := c.Param("id")

	err := m.meterService.Delete(c.Request.Context(), orgId, meterId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.Status(204)
}