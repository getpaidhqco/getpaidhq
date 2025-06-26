package pdf

import (
	"os"
	"path/filepath"
	"payloop/internal/domain/entities"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTemplateEngine is a mock implementation of the TemplateEngine interface
type MockTemplateEngine struct {
	mock.Mock
}

func (m *MockTemplateEngine) ParseAndRenderString(template string, data interface{}) (string, error) {
	args := m.Called(template, data)
	return args.String(0), args.Error(1)
}

// MockFileSystem is a mock implementation of the FileSystem interface
type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	args := m.Called(path, data, perm)
	return args.Error(0)
}

// MockPDFEngine is a mock implementation of the PDFEngine interface
type MockPDFEngine struct {
	mock.Mock
}

func (m *MockPDFEngine) Generate(htmlContent string) ([]byte, error) {
	args := m.Called(htmlContent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// TestGenerateWithMissingTemplateName tests the Generate method with a missing template name
func TestGenerateWithMissingTemplateName(t *testing.T) {
	// Create mock dependencies
	mockTemplateEngine := new(MockTemplateEngine)
	mockFileSystem := new(MockFileSystem)
	mockPDFEngine := new(MockPDFEngine)

	// Create a PDFGenerator with the mock dependencies
	generator := NewPDFGeneratorWithDeps(mockTemplateEngine, mockFileSystem, mockPDFEngine, "templates")

	// Create test data
	invoice := createMockInvoice()
	lineItems := []entities.InvoiceLineItem{createMockLineItem()}
	options := GenerateOptions{
		TemplateName: "",
	}

	// Generate the PDF
	_, err := generator.GenerateWithLineItems(invoice, lineItems, options)

	// Verify the result
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template name is required")

	// No expectations should have been called
	mockTemplateEngine.AssertNotCalled(t, "ParseAndRenderString")
	mockFileSystem.AssertNotCalled(t, "ReadFile")
	mockPDFEngine.AssertNotCalled(t, "Generate")
}

// TestGenerateWithNonExistentTemplate tests the Generate method with a non-existent template
func TestGenerateWithNonExistentTemplate(t *testing.T) {
	// Create mock dependencies
	mockTemplateEngine := new(MockTemplateEngine)
	mockFileSystem := new(MockFileSystem)
	mockPDFEngine := new(MockPDFEngine)

	// Set up expectations
	templateName := "non_existent_template.liquid"
	mockFileSystem.On("ReadFile", filepath.Join("templates", templateName)).Return(nil, os.ErrNotExist)

	// Create a PDFGenerator with the mock dependencies
	generator := NewPDFGeneratorWithDeps(mockTemplateEngine, mockFileSystem, mockPDFEngine, "templates")

	// Create test data
	invoice := createMockInvoice()
	lineItems := []entities.InvoiceLineItem{createMockLineItem()}
	options := GenerateOptions{
		TemplateName: templateName,
	}

	// Generate the PDF
	_, err := generator.GenerateWithLineItems(invoice, lineItems, options)

	// Verify the result
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template file")

	// Verify that the expectations were met
	mockFileSystem.AssertExpectations(t)
	mockTemplateEngine.AssertNotCalled(t, "ParseAndRenderString")
	mockPDFEngine.AssertNotCalled(t, "Generate")
}

// TestPrepareTemplateData tests the prepareTemplateData method
func TestPrepareTemplateData(t *testing.T) {
	// Create mock dependencies
	mockTemplateEngine := new(MockTemplateEngine)
	mockFileSystem := new(MockFileSystem)
	mockPDFEngine := new(MockPDFEngine)

	// Create a PDFGenerator with the mock dependencies
	generator := NewPDFGeneratorWithDeps(mockTemplateEngine, mockFileSystem, mockPDFEngine, "templates")

	// Create test data
	invoice := createMockInvoice()
	lineItems := []entities.InvoiceLineItem{createMockLineItem()}

	// Call the method under test
	data, err := generator.prepareTemplateData(invoice, lineItems)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, data)

	// Verify the data contains the expected fields
	assert.Equal(t, invoice.DocNumber, data["document_number"])
	assert.Equal(t, formatDate(invoice.CreatedAt), data["create_date"])
	assert.Equal(t, formatDate(invoice.DueAt), data["due_date"])
	assert.Equal(t, formatCurrency(invoice.SubTotal), data["sub_total"])
	assert.Equal(t, formatCurrency(invoice.TaxTotal), data["tax_total"])
	assert.Equal(t, formatCurrency(invoice.Total), data["total"])
	assert.Equal(t, getCurrencySymbol(invoice.Currency), data["currency_symbol"])

	// Verify line items
	items := data["items"].([]map[string]interface{})
	assert.Len(t, items, 1)
	assert.Equal(t, lineItems[0].Description, items[0]["name"])
	assert.Equal(t, lineItems[0].Quantity, items[0]["quantity"])
	assert.Equal(t, formatCurrency(lineItems[0].UnitPrice), items[0]["unit_price"])
	assert.Equal(t, formatCurrency(lineItems[0].LineTotal), items[0]["total"])
	assert.Equal(t, formatCurrency(lineItems[0].TaxAmount), items[0]["tax_total"])

	// Verify tax data
	tax := items[0]["tax"].(map[string]interface{})
	assert.Equal(t, lineItems[0].TaxCode, tax["name"])
	assert.Equal(t, formatTaxRate(lineItems[0].TaxRate), tax["rate"])
}

// TestFormatCurrency tests the formatCurrency function
func TestFormatCurrency(t *testing.T) {
	testCases := []struct {
		amount   int
		expected string
	}{
		{1000, "10.00"},
		{1250, "12.50"},
		{0, "0.00"},
		{-500, "-5.00"},
	}

	for _, tc := range testCases {
		result := formatCurrency(tc.amount)
		assert.Equal(t, tc.expected, result)
	}
}

// TestFormatTaxRate tests the formatTaxRate function
func TestFormatTaxRate(t *testing.T) {
	testCases := []struct {
		rate     int
		expected string
	}{
		{1000, "10.00"},
		{750, "7.50"},
		{0, "0.00"},
		{2500, "25.00"},
	}

	for _, tc := range testCases {
		result := formatTaxRate(tc.rate)
		assert.Equal(t, tc.expected, result)
	}
}

// TestFormatDate tests the formatDate function
func TestFormatDate(t *testing.T) {
	now := time.Now()
	result := formatDate(now)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, now.String())

	// Test with nil
	result = formatDate(nil)
	assert.Equal(t, "<nil>", result)
}

// TestGetCurrencySymbol tests the getCurrencySymbol function
func TestGetCurrencySymbol(t *testing.T) {
	testCases := []struct {
		currency string
		expected string
	}{
		{"USD", "$"},
		{"EUR", "€"},
		{"GBP", "£"},
		{"XYZ", "XYZ"}, // Unknown currency should return the currency code
	}

	for _, tc := range testCases {
		result := getCurrencySymbol(tc.currency)
		assert.Equal(t, tc.expected, result)
	}
}

// Helper functions to create mock data for testing

func createMockInvoice() entities.Invoice {
	return entities.Invoice{
		OrgId:         "org123",
		Id:            "inv123",
		CustomerId:    "cus123",
		DocNumber:     "INV-001",
		Type:          entities.DocumentTypeInvoice,
		InvoiceType:   entities.InvoiceTypeInitial,
		Status:        entities.InvoiceStatusDraft,
		Currency:      "USD",
		SubTotal:      1000,
		TaxTotal:      100,
		DiscountTotal: 0,
		Total:         1100,
		AmountPaid:    0,
		AmountDue:     1100,
		DueAt:         time.Now().Add(30 * 24 * time.Hour),
		Notes:         "Test invoice",
		CustomerNotes: "Thank you for your business",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func createMockLineItem() entities.InvoiceLineItem {
	return entities.InvoiceLineItem{
		OrgId:       "org123",
		InvoiceId:   "inv123",
		Id:          "line123",
		Description: "Test product",
		Quantity:    1,
		UnitPrice:   1000,
		LineTotal:   1000,
		TaxCode:     "TAX",
		TaxRate:     10,
		TaxAmount:   100,
		TaxExempt:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// TestGenerateWithMocks tests the Generate method with mocked dependencies
func TestGenerateWithMocks(t *testing.T) {
	// Set up test data
	tempDir := t.TempDir()
	templateName := "test_template.liquid"
	templateContent := "<html><body>Test Template</body></html>"

	// Create mock dependencies
	mockTemplateEngine := new(MockTemplateEngine)
	mockFileSystem := new(MockFileSystem)
	mockPDFEngine := new(MockPDFEngine)

	// Set up expectations
	mockFileSystem.On("ReadFile", filepath.Join(tempDir, templateName)).Return([]byte(templateContent), nil)
	mockTemplateEngine.On("ParseAndRenderString", string(templateContent), mock.Anything).Return("<html><body>Rendered Template</body></html>", nil)
	mockPDFEngine.On("Generate", "<html><body>Rendered Template</body></html>").Return([]byte("PDF Content"), nil)
	mockFileSystem.On("WriteFile", mock.AnythingOfType("string"), []byte("PDF Content"), os.FileMode(0644)).Return(nil)

	// Create a PDFGenerator with the mock dependencies
	generator := NewPDFGeneratorWithDeps(mockTemplateEngine, mockFileSystem, mockPDFEngine, tempDir)

	// Create test data
	invoice := createMockInvoice()
	lineItems := []entities.InvoiceLineItem{createMockLineItem()}
	options := GenerateOptions{
		TemplateName: templateName,
	}

	// Generate the PDF
	pdfBytes, err := generator.GenerateWithLineItems(invoice, lineItems, options)

	// Verify the result
	assert.NoError(t, err)
	assert.Equal(t, []byte("PDF Content"), pdfBytes)

	// Verify that all expectations were met
	mockTemplateEngine.AssertExpectations(t)
	mockFileSystem.AssertExpectations(t)
	mockPDFEngine.AssertExpectations(t)
}

// TestGenerateWithOutputPath tests the Generate method with an output path
func TestGenerateWithOutputPath(t *testing.T) {
	// Set up test data
	tempDir := t.TempDir()
	templateName := "test_template.liquid"
	templateContent := "<html><body>Test Template</body></html>"
	outputPath := filepath.Join(tempDir, "output.pdf")

	// Create mock dependencies
	mockTemplateEngine := new(MockTemplateEngine)
	mockFileSystem := new(MockFileSystem)
	mockPDFEngine := new(MockPDFEngine)

	// Set up expectations
	mockFileSystem.On("ReadFile", filepath.Join(tempDir, templateName)).Return([]byte(templateContent), nil)
	mockTemplateEngine.On("ParseAndRenderString", string(templateContent), mock.Anything).Return("<html><body>Rendered Template</body></html>", nil)
	mockPDFEngine.On("Generate", "<html><body>Rendered Template</body></html>").Return([]byte("PDF Content"), nil)
	mockFileSystem.On("WriteFile", outputPath, []byte("PDF Content"), os.FileMode(0644)).Return(nil)

	// Create a PDFGenerator with the mock dependencies
	generator := NewPDFGeneratorWithDeps(mockTemplateEngine, mockFileSystem, mockPDFEngine, tempDir)

	// Create test data
	invoice := createMockInvoice()
	lineItems := []entities.InvoiceLineItem{createMockLineItem()}
	options := GenerateOptions{
		TemplateName: templateName,
		OutputPath:   outputPath,
	}

	// Generate the PDF
	pdfBytes, err := generator.GenerateWithLineItems(invoice, lineItems, options)

	// Verify the result
	assert.NoError(t, err)
	assert.Equal(t, []byte("PDF Content"), pdfBytes)

	// Verify that all expectations were met
	mockTemplateEngine.AssertExpectations(t)
	mockFileSystem.AssertExpectations(t)
	mockPDFEngine.AssertExpectations(t)
}
