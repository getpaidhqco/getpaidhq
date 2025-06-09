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

type InvoiceController struct {
	invoiceService interfaces.InvoiceService
	logger         logger.Logger
}

func NewInvoiceController(invoiceService interfaces.InvoiceService, logger logger.Logger) InvoiceController {
	return InvoiceController{
		invoiceService: invoiceService,
		logger:         logger,
	}
}

// Create handles the creation of a new invoice
func (c InvoiceController) Create(ctx *gin.Context) {
	var input request.CreateInvoiceRequest
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)

	if err := ctx.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	invoice, err := c.invoiceService.Create(ctx.Request.Context(), authUser.OrgId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Get line items for the invoice
	lineItems, err := c.invoiceService.ListLineItems(ctx.Request.Context(), authUser.OrgId, invoice.Id)
	if err != nil {
		c.logger.Error("Failed to get line items for invoice: ", err)
		// Continue even if line items retrieval fails
	}

	// Convert line items to response DTOs
	lineItemResponses := make([]response.InvoiceLineItem, len(lineItems))
	for i, item := range lineItems {
		lineItemResponses[i] = response.NewInvoiceLineItemFromEntity(item)
	}

	// Create response DTO
	invoiceResponse := response.NewInvoiceFromEntity(invoice)
	invoiceResponse.LineItems = lineItemResponses

	ctx.JSON(http.StatusOK, invoiceResponse)
}

// Get handles retrieving an invoice by ID
func (c InvoiceController) Get(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	invoiceId := ctx.Param("id")

	invoice, err := c.invoiceService.Get(ctx.Request.Context(), authUser.OrgId, invoiceId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Get line items for the invoice
	lineItems, err := c.invoiceService.ListLineItems(ctx.Request.Context(), authUser.OrgId, invoice.Id)
	if err != nil {
		c.logger.Error("Failed to get line items for invoice: ", err)
		// Continue even if line items retrieval fails
	}

	// Convert line items to response DTOs
	lineItemResponses := make([]response.InvoiceLineItem, len(lineItems))
	for i, item := range lineItems {
		lineItemResponses[i] = response.NewInvoiceLineItemFromEntity(item)
	}

	// Create response DTO
	invoiceResponse := response.NewInvoiceFromEntity(invoice)
	invoiceResponse.LineItems = lineItemResponses

	ctx.JSON(http.StatusOK, invoiceResponse)
}

// Update handles updating an existing invoice
func (c InvoiceController) Update(ctx *gin.Context) {
	var input request.UpdateInvoiceRequest
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	invoiceId := ctx.Param("id")

	if err := ctx.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	invoice, err := c.invoiceService.Update(ctx.Request.Context(), authUser.OrgId, invoiceId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Get line items for the invoice
	lineItems, err := c.invoiceService.ListLineItems(ctx.Request.Context(), authUser.OrgId, invoice.Id)
	if err != nil {
		c.logger.Error("Failed to get line items for invoice: ", err)
		// Continue even if line items retrieval fails
	}

	// Convert line items to response DTOs
	lineItemResponses := make([]response.InvoiceLineItem, len(lineItems))
	for i, item := range lineItems {
		lineItemResponses[i] = response.NewInvoiceLineItemFromEntity(item)
	}

	// Create response DTO
	invoiceResponse := response.NewInvoiceFromEntity(invoice)
	invoiceResponse.LineItems = lineItemResponses

	ctx.JSON(http.StatusOK, invoiceResponse)
}

// List handles retrieving a list of invoices
func (c InvoiceController) List(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	pagination := request.GetPagination(ctx)

	invoices, total, err := c.invoiceService.List(ctx.Request.Context(), authUser.OrgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert invoices to response DTOs
	invoiceResponses := make([]response.Invoice, len(invoices))
	for i, invoice := range invoices {
		invoiceResponses[i] = response.NewInvoiceFromEntity(invoice)
	}

	ctx.JSON(http.StatusOK, response.ListResponse{
		Data: invoiceResponses,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// ListByCustomer handles retrieving a list of invoices for a specific customer
func (c InvoiceController) ListByCustomer(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	customerId := ctx.Param("id")
	pagination := request.GetPagination(ctx)

	invoices, total, err := c.invoiceService.FindByCustomerId(ctx.Request.Context(), authUser.OrgId, customerId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert invoices to response DTOs
	invoiceResponses := make([]response.Invoice, len(invoices))
	for i, invoice := range invoices {
		invoiceResponses[i] = response.NewInvoiceFromEntity(invoice)
	}

	ctx.JSON(http.StatusOK, response.ListResponse{
		Data: invoiceResponses,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// PerformAction handles performing an action on an invoice
func (c InvoiceController) PerformAction(ctx *gin.Context) {
	var input request.InvoiceActionRequest
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	invoiceId := ctx.Param("id")

	if err := ctx.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	invoice, err := c.invoiceService.PerformAction(ctx.Request.Context(), authUser.OrgId, invoiceId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Get line items for the invoice
	lineItems, err := c.invoiceService.ListLineItems(ctx.Request.Context(), authUser.OrgId, invoice.Id)
	if err != nil {
		c.logger.Error("Failed to get line items for invoice: ", err)
		// Continue even if line items retrieval fails
	}

	// Convert line items to response DTOs
	lineItemResponses := make([]response.InvoiceLineItem, len(lineItems))
	for i, item := range lineItems {
		lineItemResponses[i] = response.NewInvoiceLineItemFromEntity(item)
	}

	// Create response DTO
	invoiceResponse := response.NewInvoiceFromEntity(invoice)
	invoiceResponse.LineItems = lineItemResponses

	ctx.JSON(http.StatusOK, invoiceResponse)
}

// AddLineItem handles adding a line item to an invoice
func (c InvoiceController) AddLineItem(ctx *gin.Context) {
	var input request.CreateInvoiceLineItemRequest
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	invoiceId := ctx.Param("id")

	if err := ctx.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	lineItem, err := c.invoiceService.AddLineItem(ctx.Request.Context(), authUser.OrgId, invoiceId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.JSON(http.StatusOK, response.NewInvoiceLineItemFromEntity(lineItem))
}

// UpdateLineItem handles updating a line item in an invoice
func (c InvoiceController) UpdateLineItem(ctx *gin.Context) {
	var input request.UpdateInvoiceLineItemRequest
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	invoiceId := ctx.Param("id")
	lineItemId := ctx.Param("lineItemId")

	if err := ctx.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	lineItem, err := c.invoiceService.UpdateLineItem(ctx.Request.Context(), authUser.OrgId, invoiceId, lineItemId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.JSON(http.StatusOK, response.NewInvoiceLineItemFromEntity(lineItem))
}

// DeleteLineItem handles deleting a line item from an invoice
func (c InvoiceController) DeleteLineItem(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	invoiceId := ctx.Param("id")
	lineItemId := ctx.Param("lineItemId")

	err := c.invoiceService.DeleteLineItem(ctx.Request.Context(), authUser.OrgId, invoiceId, lineItemId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

// ListLineItems handles retrieving a list of line items for an invoice
func (c InvoiceController) ListLineItems(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	invoiceId := ctx.Param("id")

	lineItems, err := c.invoiceService.ListLineItems(ctx.Request.Context(), authUser.OrgId, invoiceId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert line items to response DTOs
	lineItemResponses := make([]response.InvoiceLineItem, len(lineItems))
	for i, item := range lineItems {
		lineItemResponses[i] = response.NewInvoiceLineItemFromEntity(item)
	}

	ctx.JSON(http.StatusOK, lineItemResponses)
}

// ListHistory handles retrieving the history of an invoice
func (c InvoiceController) ListHistory(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	authUser := user.(authn.User)
	invoiceId := ctx.Param("id")

	history, err := c.invoiceService.ListHistory(ctx.Request.Context(), authUser.OrgId, invoiceId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		ctx.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert history entries to response DTOs
	historyResponses := make([]response.InvoiceHistory, len(history))
	for i, entry := range history {
		historyResponses[i] = response.NewInvoiceHistoryFromEntity(entry)
	}

	ctx.JSON(http.StatusOK, historyResponses)
}
