package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockInvoiceService is a mock implementation of the InvoiceService interface
type MockInvoiceService struct {
	mock.Mock
}

// Create mocks the Create method of InvoiceService
func (m *MockInvoiceService) Create(ctx context.Context, orgId string, req request.CreateInvoiceRequest) (entities.Invoice, error) {
	args := m.Called(ctx, orgId, req)
	return args.Get(0).(entities.Invoice), args.Error(1)
}

// Get mocks the Get method of InvoiceService
func (m *MockInvoiceService) Get(ctx context.Context, orgId string, id string) (entities.Invoice, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Invoice), args.Error(1)
}

// Update mocks the Update method of InvoiceService
func (m *MockInvoiceService) Update(ctx context.Context, orgId string, id string, req request.UpdateInvoiceRequest) (entities.Invoice, error) {
	args := m.Called(ctx, orgId, id, req)
	return args.Get(0).(entities.Invoice), args.Error(1)
}

// List mocks the List method of InvoiceService
func (m *MockInvoiceService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Invoice, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]entities.Invoice), args.Int(1), args.Error(2)
}

// FindByCustomerId mocks the FindByCustomerId method of InvoiceService
func (m *MockInvoiceService) FindByCustomerId(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]entities.Invoice, int, error) {
	args := m.Called(ctx, orgId, customerId, pagination)
	return args.Get(0).([]entities.Invoice), args.Int(1), args.Error(2)
}

// PerformAction mocks the PerformAction method of InvoiceService
func (m *MockInvoiceService) PerformAction(ctx context.Context, orgId string, id string, req request.InvoiceActionRequest) (entities.Invoice, error) {
	args := m.Called(ctx, orgId, id, req)
	return args.Get(0).(entities.Invoice), args.Error(1)
}

// AddLineItem mocks the AddLineItem method of InvoiceService
func (m *MockInvoiceService) AddLineItem(ctx context.Context, orgId string, invoiceId string, req request.CreateInvoiceLineItemRequest) (entities.InvoiceLineItem, error) {
	args := m.Called(ctx, orgId, invoiceId, req)
	return args.Get(0).(entities.InvoiceLineItem), args.Error(1)
}

// UpdateLineItem mocks the UpdateLineItem method of InvoiceService
func (m *MockInvoiceService) UpdateLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string, req request.UpdateInvoiceLineItemRequest) (entities.InvoiceLineItem, error) {
	args := m.Called(ctx, orgId, invoiceId, lineItemId, req)
	return args.Get(0).(entities.InvoiceLineItem), args.Error(1)
}

// DeleteLineItem mocks the DeleteLineItem method of InvoiceService
func (m *MockInvoiceService) DeleteLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string) error {
	args := m.Called(ctx, orgId, invoiceId, lineItemId)
	return args.Error(0)
}

// ListLineItems mocks the ListLineItems method of InvoiceService
func (m *MockInvoiceService) ListLineItems(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceLineItem, error) {
	args := m.Called(ctx, orgId, invoiceId)
	return args.Get(0).([]entities.InvoiceLineItem), args.Error(1)
}

// ListHistory mocks the ListHistory method of InvoiceService
func (m *MockInvoiceService) ListHistory(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceHistory, error) {
	args := m.Called(ctx, orgId, invoiceId)
	return args.Get(0).([]entities.InvoiceHistory), args.Error(1)
}

// MockLogger is a mock implementation of the logger.Logger interface
type MockLogger struct {
	mock.Mock
}

var _ logger.Logger = (*MockLogger)(nil) // Ensure MockLogger implements logger.Logger

func (m *MockLogger) Debug(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Warn(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Fatal(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Debugf(template string, args ...interface{}) {
	m.Called(template, args)
}

func (m *MockLogger) Infof(template string, args ...interface{}) {
	m.Called(template, args)
}

func (m *MockLogger) Warnf(template string, args ...interface{}) {
	m.Called(template, args)
}

func (m *MockLogger) Errorf(template string, args ...interface{}) {
	m.Called(template, args)
}

func (m *MockLogger) Panicf(template string, args ...interface{}) {
	m.Called(template, args)
}

func (m *MockLogger) Fatalf(template string, args ...interface{}) {
	m.Called(template, args)
}

func (m *MockLogger) Sync() error {
	args := m.Called()
	return args.Error(0)
}

// setupTestRouter sets up a test router with the invoice controller
func setupTestRouter(mockService *MockInvoiceService, mockLogger *MockLogger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create the controller with the mock service and logger
	controller := NewInvoiceController(mockService, mockLogger)

	// Add a middleware to set the user in the context
	router.Use(func(c *gin.Context) {
		c.Set("user", authn.User{
			Id:          "user123",
			OrgId:       "org123",
			Email:       "test@example.com",
			PrimaryRole: authn.Admin,
			Roles:       []authn.UserRole{authn.Admin},
		})
		c.Next()
	})

	// Set up the routes
	router.POST("/invoices", controller.Create)
	router.GET("/invoices/:id", controller.Get)
	router.PUT("/invoices/:id", controller.Update)
	router.GET("/invoices", controller.List)
	router.GET("/customers/:id/invoices", controller.ListByCustomer)
	router.POST("/invoices/:id/actions", controller.PerformAction)
	router.POST("/invoices/:id/line-items", controller.AddLineItem)
	router.PUT("/invoices/:id/line-items/:lineItemId", controller.UpdateLineItem)
	router.DELETE("/invoices/:id/line-items/:lineItemId", controller.DeleteLineItem)
	router.GET("/invoices/:id/line-items", controller.ListLineItems)
	router.GET("/invoices/:id/history", controller.ListHistory)

	return router
}

// createMockInvoice creates a mock invoice for testing
func createMockInvoice() entities.Invoice {
	return entities.Invoice{
		OrgId:          "org123",
		Id:             "inv123",
		CustomerId:     "cus123",
		OrderId:        "ord123",
		SubscriptionId: "sub123",
		SequenceId:     "seq123",
		DocNumber:      "INV-001",
		Type:           entities.DocumentTypeInvoice,
		InvoiceType:    entities.InvoiceTypeInitial,
		Status:         entities.InvoiceStatusDraft,
		IsImmutable:    false,
		Currency:       "USD",
		SubTotal:       1000,
		TaxTotal:       100,
		DiscountTotal:  0,
		Total:          1100,
		AmountPaid:     0,
		AmountDue:      1100,
		DueAt:          time.Now().Add(30 * 24 * time.Hour),
		Notes:          "Test invoice",
		CustomerNotes:  "Thank you for your business",
		Metadata:       map[string]string{"test": "value"},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// createMockLineItem creates a mock line item for testing
func createMockLineItem() entities.InvoiceLineItem {
	return entities.InvoiceLineItem{
		OrgId:         "org123",
		InvoiceId:     "inv123",
		Id:            "line123",
		ProductId:     "prod123",
		VariantId:     "var123",
		PriceId:       "price123",
		Description:   "Test product",
		Category:      "Test category",
		Quantity:      1,
		UnitPrice:     1000,
		LineTotal:     1000,
		DiscountType:  "",
		DiscountValue: 0,
		DiscountTotal: 0,
		TaxCode:       "tax123",
		TaxRate:       10,
		TaxAmount:     100,
		TaxExempt:     false,
		Seq:           1,
		Metadata:      map[string]string{"test": "value"},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// createMockHistory creates a mock history entry for testing
func createMockHistory() entities.InvoiceHistory {
	return entities.InvoiceHistory{
		OrgId:     "org123",
		Id:        "hist123",
		InvoiceId: "inv123",
		Action:    entities.InvoiceHistoryActionCreated,
		UserId:    "user123",
		UserEmail: "test@example.com",
		Timestamp: time.Now(),
	}
}

// TestCreateInvoice tests the Create method of the invoice controller
func TestCreateInvoice(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create a mock invoice request
	createRequest := request.CreateInvoiceRequest{
		CustomerId:  "cus123",
		OrderId:     "ord123",
		Type:        entities.DocumentTypeInvoice,
		InvoiceType: entities.InvoiceTypeInitial,
		Currency:    "USD",
		DueAt:       time.Now().Add(30 * 24 * time.Hour),
		Notes:       "Test invoice",
		LineItems: []request.CreateInvoiceLineItemRequest{
			{
				ProductId:   "prod123",
				Description: "Test product",
				Quantity:    1,
				UnitPrice:   1000,
			},
		},
	}

	// Create mock invoice and line items
	mockInvoice := createMockInvoice()
	mockLineItems := []entities.InvoiceLineItem{createMockLineItem()}

	// Set up expectations
	mockService.On("Create", mock.Anything, "org123", mock.AnythingOfType("request.CreateInvoiceRequest")).Return(mockInvoice, nil)
	mockService.On("ListLineItems", mock.Anything, "org123", "inv123").Return(mockLineItems, nil)
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	// Create a request body
	requestBody, _ := json.Marshal(createRequest)

	// Create a test request
	req, _ := http.NewRequest("POST", "/invoices", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestGetInvoice tests the Get method of the invoice controller
func TestGetInvoice(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create mock invoice and line items
	mockInvoice := createMockInvoice()
	mockLineItems := []entities.InvoiceLineItem{createMockLineItem()}

	// Set up expectations
	mockService.On("Get", mock.Anything, "org123", "inv123").Return(mockInvoice, nil)
	mockService.On("ListLineItems", mock.Anything, "org123", "inv123").Return(mockLineItems, nil)
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	// Create a test request
	req, _ := http.NewRequest("GET", "/invoices/inv123", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestListInvoices tests the List method of the invoice controller
func TestListInvoices(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create mock invoices
	mockInvoices := []entities.Invoice{createMockInvoice()}

	// Set up expectations
	mockService.On("List", mock.Anything, "org123", mock.AnythingOfType("request.Pagination")).Return(mockInvoices, 1, nil)

	// Create a test request
	req, _ := http.NewRequest("GET", "/invoices", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestUpdateInvoice tests the Update method of the invoice controller
func TestUpdateInvoice(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create a mock invoice update request
	updateRequest := request.UpdateInvoiceRequest{
		Notes:         "Updated notes",
		CustomerNotes: "Updated customer notes",
		DueAt:         time.Now().Add(60 * 24 * time.Hour),
		Metadata:      map[string]string{"updated": "value"},
	}

	// Create mock invoice and line items
	mockInvoice := createMockInvoice()
	mockLineItems := []entities.InvoiceLineItem{createMockLineItem()}

	// Set up expectations
	mockService.On("Update", mock.Anything, "org123", "inv123", mock.AnythingOfType("request.UpdateInvoiceRequest")).Return(mockInvoice, nil)
	mockService.On("ListLineItems", mock.Anything, "org123", "inv123").Return(mockLineItems, nil)
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	// Create a request body
	requestBody, _ := json.Marshal(updateRequest)

	// Create a test request
	req, _ := http.NewRequest("PUT", "/invoices/inv123", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestListByCustomer tests the ListByCustomer method of the invoice controller
func TestListByCustomer(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create mock invoices
	mockInvoices := []entities.Invoice{createMockInvoice()}

	// Set up expectations
	mockService.On("FindByCustomerId", mock.Anything, "org123", "cus123", mock.AnythingOfType("request.Pagination")).Return(mockInvoices, 1, nil)

	// Create a test request
	req, _ := http.NewRequest("GET", "/customers/cus123/invoices", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestPerformAction tests the PerformAction method of the invoice controller
func TestPerformAction(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create a mock invoice action request
	actionRequest := request.InvoiceActionRequest{
		Action: "send",
		Reason: "Test reason",
	}

	// Create mock invoice and line items
	mockInvoice := createMockInvoice()
	mockLineItems := []entities.InvoiceLineItem{createMockLineItem()}

	// Set up expectations
	mockService.On("PerformAction", mock.Anything, "org123", "inv123", mock.AnythingOfType("request.InvoiceActionRequest")).Return(mockInvoice, nil)
	mockService.On("ListLineItems", mock.Anything, "org123", "inv123").Return(mockLineItems, nil)
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	// Create a request body
	requestBody, _ := json.Marshal(actionRequest)

	// Create a test request
	req, _ := http.NewRequest("POST", "/invoices/inv123/actions", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestAddLineItem tests the AddLineItem method of the invoice controller
func TestAddLineItem(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create a mock line item request
	lineItemRequest := request.CreateInvoiceLineItemRequest{
		ProductId:   "prod123",
		Description: "Test product",
		Quantity:    1,
		UnitPrice:   1000,
	}

	// Create mock line item
	mockLineItem := createMockLineItem()

	// Set up expectations
	mockService.On("AddLineItem", mock.Anything, "org123", "inv123", mock.AnythingOfType("request.CreateInvoiceLineItemRequest")).Return(mockLineItem, nil)

	// Create a request body
	requestBody, _ := json.Marshal(lineItemRequest)

	// Create a test request
	req, _ := http.NewRequest("POST", "/invoices/inv123/line-items", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestUpdateLineItem tests the UpdateLineItem method of the invoice controller
func TestUpdateLineItem(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create a mock line item update request
	lineItemUpdateRequest := request.UpdateInvoiceLineItemRequest{
		Description: "Updated product",
		Quantity:    2,
		UnitPrice:   2000,
	}

	// Create mock line item
	mockLineItem := createMockLineItem()

	// Set up expectations
	mockService.On("UpdateLineItem", mock.Anything, "org123", "inv123", "line123", mock.AnythingOfType("request.UpdateInvoiceLineItemRequest")).Return(mockLineItem, nil)

	// Create a request body
	requestBody, _ := json.Marshal(lineItemUpdateRequest)

	// Create a test request
	req, _ := http.NewRequest("PUT", "/invoices/inv123/line-items/line123", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestDeleteLineItem tests the DeleteLineItem method of the invoice controller
func TestDeleteLineItem(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Set up expectations
	mockService.On("DeleteLineItem", mock.Anything, "org123", "inv123", "line123").Return(nil)

	// Create a test request
	req, _ := http.NewRequest("DELETE", "/invoices/inv123/line-items/line123", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestListLineItems tests the ListLineItems method of the invoice controller
func TestListLineItems(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create mock line items
	mockLineItems := []entities.InvoiceLineItem{createMockLineItem()}

	// Set up expectations
	mockService.On("ListLineItems", mock.Anything, "org123", "inv123").Return(mockLineItems, nil)

	// Create a test request
	req, _ := http.NewRequest("GET", "/invoices/inv123/line-items", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}

// TestListHistory tests the ListHistory method of the invoice controller
func TestListHistory(t *testing.T) {
	// Create mock service and logger
	mockService := new(MockInvoiceService)
	mockLogger := new(MockLogger)

	// Set up the test router
	router := setupTestRouter(mockService, mockLogger)

	// Create mock history entries
	mockHistory := []entities.InvoiceHistory{createMockHistory()}

	// Set up expectations
	mockService.On("ListHistory", mock.Anything, "org123", "inv123").Return(mockHistory, nil)

	// Create a test request
	req, _ := http.NewRequest("GET", "/invoices/inv123/history", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that the expectations were met
	mockService.AssertExpectations(t)
}
