package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
)

// SettingsController handles HTTP requests for settings
type SettingsController struct {
	settingsService interfaces.SettingsService
	logger          logger.Logger
}

// NewSettingsController creates a new SettingsController
func NewSettingsController(settingsService interfaces.SettingsService, logger logger.Logger) SettingsController {
	return SettingsController{
		settingsService: settingsService,
		logger:          logger,
	}
}

// Get retrieves a setting by ID
func (s SettingsController) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	parentId := c.Param("parent_id")
	id := c.Param("id")

	settings, err := s.settingsService.GetSettingRaw(c.Request.Context(), orgId, parentId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, settings)
}

// List retrieves all settings for a parent
func (s SettingsController) List(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	parentId := c.Param("parent_id")

	settings, err := s.settingsService.ListSettings(c.Request.Context(), orgId, parentId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, response.ListResponse{
		Data: response.NewSettingsFromEntities(settings),
		Meta: response.Meta{
			Total: len(settings),
		},
	})
}

// Create creates a new setting or updates it if it already exists
func (s SettingsController) Create(c *gin.Context) {
	var input request.UpsertSettingRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	setting, err := s.settingsService.UpsertSetting(c.Request.Context(), orgId, input.ParentId, input.Id, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusCreated, response.NewSettingFromEntity(setting))
}

// Update updates an existing setting by merging values
func (s SettingsController) Update(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	parentId := c.Param("parent_id")
	id := c.Param("id")

	var input interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	setting, err := s.settingsService.UpsertSetting(c.Request.Context(), orgId, parentId, id, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, setting.Value)
}

// Delete deletes a setting
func (s SettingsController) Delete(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	parentId := c.Param("parent_id")
	id := c.Param("id")

	err := s.settingsService.DeleteSetting(c.Request.Context(), orgId, parentId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
