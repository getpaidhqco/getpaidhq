package pdf

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"payloop/internal/domain/entities"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/osteele/liquid"
)

// TemplateDir is the directory where invoice templates are stored
const TemplateDir = "assets/templates/invoices"

// TemplateEngine is an interface for template rendering engines
type TemplateEngine interface {
	ParseAndRenderString(template string, data interface{}) (string, error)
}

// LiquidTemplateEngine is an adapter for liquid.Engine that implements TemplateEngine
type LiquidTemplateEngine struct {
	engine *liquid.Engine
}

// NewLiquidTemplateEngine creates a new LiquidTemplateEngine
func NewLiquidTemplateEngine() *LiquidTemplateEngine {
	return &LiquidTemplateEngine{
		engine: liquid.NewEngine(),
	}
}

// ParseAndRenderString renders a template with the given data
func (e *LiquidTemplateEngine) ParseAndRenderString(template string, data interface{}) (string, error) {
	// Convert data to liquid.Bindings if it's a map
	var bindings liquid.Bindings
	if m, ok := data.(map[string]interface{}); ok {
		bindings = liquid.Bindings(m)
	} else {
		// If it's not a map, return an error
		return "", fmt.Errorf("data must be a map[string]interface{}, got %T", data)
	}

	result, err := e.engine.ParseAndRenderString(template, bindings)
	if err != nil {
		return "", err
	}
	return result, nil
}

// FileSystem is an interface for file system operations
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
}

// DefaultFileSystem is the default implementation of FileSystem
type DefaultFileSystem struct{}

// ReadFile reads a file from the file system
func (fs DefaultFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file in the file system
func (fs DefaultFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// PDFEngine is an interface for PDF generation engines
type PDFEngine interface {
	Generate(htmlContent string) ([]byte, error)
}

// WkhtmltopdfEngine is an implementation of PDFEngine using wkhtmltopdf
type WkhtmltopdfEngine struct{}

// Generate generates a PDF from HTML content using wkhtmltopdf
func (e WkhtmltopdfEngine) Generate(htmlContent string) ([]byte, error) {
	// Create a new PDF generator
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	// Set options
	pdfg.Dpi.Set(300)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.Grayscale.Set(false)

	// Add HTML content
	page := wkhtmltopdf.NewPageReader(bytes.NewReader([]byte(htmlContent)))
	page.EnableLocalFileAccess.Set(true)
	pdfg.AddPage(page)

	// Create PDF
	err = pdfg.Create()
	if err != nil {
		return nil, err
	}

	// Get PDF bytes
	return pdfg.Bytes(), nil
}

// PDFGenerator is a utility for generating PDF documents from templates
type PDFGenerator struct {
	templateEngine TemplateEngine
	fileSystem     FileSystem
	pdfEngine      PDFEngine
	templateDir    string
}

// NewPDFGenerator creates a new PDFGenerator instance with default dependencies
func NewPDFGenerator() *PDFGenerator {
	templateEngine := NewLiquidTemplateEngine()
	return &PDFGenerator{
		templateEngine: templateEngine,
		fileSystem:     DefaultFileSystem{},
		pdfEngine:      WkhtmltopdfEngine{},
		templateDir:    TemplateDir,
	}
}

// NewPDFGeneratorWithDeps creates a new PDFGenerator instance with custom dependencies
func NewPDFGeneratorWithDeps(templateEngine TemplateEngine, fileSystem FileSystem, pdfEngine PDFEngine, templateDir string) *PDFGenerator {
	return &PDFGenerator{
		templateEngine: templateEngine,
		fileSystem:     fileSystem,
		pdfEngine:      pdfEngine,
		templateDir:    templateDir,
	}
}

// GenerateOptions contains options for PDF generation
type GenerateOptions struct {
	TemplateName string
	OutputPath   string // Optional: if provided, the PDF will be saved to this path
}

// Generate generates a PDF from an invoice using the specified template
func (g *PDFGenerator) Generate(invoice entities.Invoice, lineItems []entities.InvoiceLineItem, options GenerateOptions) ([]byte, error) {
	// Validate options
	if options.TemplateName == "" {
		return nil, fmt.Errorf("template name is required")
	}

	// Load the template
	templatePath := filepath.Join(g.templateDir, options.TemplateName)
	templateContent, err := g.fileSystem.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	// Prepare template data
	templateData, err := g.prepareTemplateData(invoice, lineItems)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare template data: %w", err)
	}

	// Render the template
	renderedHTML, err := g.templateEngine.ParseAndRenderString(string(templateContent), templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Generate PDF
	pdfBytes, err := g.pdfEngine.Generate(renderedHTML)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Save to file if output path is provided
	if options.OutputPath != "" {
		err = g.fileSystem.WriteFile(options.OutputPath, pdfBytes, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write PDF to file: %w", err)
		}
	}

	return pdfBytes, nil
}

// prepareTemplateData prepares the data to be passed to the template
func (g *PDFGenerator) prepareTemplateData(invoice entities.Invoice, lineItems []entities.InvoiceLineItem) (map[string]interface{}, error) {
	// Convert line items to template format
	items := make([]map[string]interface{}, len(lineItems))
	for i, item := range lineItems {
		items[i] = map[string]interface{}{
			"name":       item.Description,
			"quantity":   item.Quantity,
			"unit_price": formatCurrency(item.UnitPrice),
			"total":      formatCurrency(item.LineTotal),
			"tax_total":  formatCurrency(item.TaxAmount),
			"tax": map[string]interface{}{
				"name": item.TaxCode,
				"rate": formatTaxRate(item.TaxRate),
			},
		}
	}

	// Prepare the data for the template
	data := map[string]interface{}{
		"document_number": invoice.DocNumber,
		"create_date":     formatDate(invoice.CreatedAt),
		"due_date":        formatDate(invoice.DueAt),
		"issued_date":     formatDate(invoice.IssuedAt),
		"paid_date":       formatDate(invoice.PaidAt),
		"currency_symbol": getCurrencySymbol(invoice.Currency),
		"currency":        invoice.Currency,
		"sub_total":       formatCurrency(invoice.SubTotal),
		"tax_total":       formatCurrency(invoice.TaxTotal),
		"discount_total":  formatCurrency(invoice.DiscountTotal),
		"total":           formatCurrency(invoice.Total),
		"amount_paid":     formatCurrency(invoice.AmountPaid),
		"amount_due":      formatCurrency(invoice.AmountDue),
		"notes":           invoice.Notes,
		"customer_notes":  invoice.CustomerNotes,
		"items":           items,
		"type":            string(invoice.Type),
		// These fields would need to be populated from other sources
		// "business": businessData,
		// "customer": customerData,
		// "logo_url": logoURL,
		// "payments": paymentsData,
		// "footer_text": footerText,
	}

	return data, nil
}


// Helper functions for formatting data

func formatCurrency(amount int) string {
	return fmt.Sprintf("%.2f", float64(amount)/100)
}

func formatTaxRate(rate int) string {
	return fmt.Sprintf("%.2f", float64(rate)/100)
}

func formatDate(date interface{}) string {
	// Format date based on the type
	// This is a simplified implementation
	return fmt.Sprintf("%v", date)
}

func getCurrencySymbol(currency string) string {
	// Map currency codes to symbols
	symbols := map[string]string{
		"USD": "$",
		"EUR": "€",
		"GBP": "£",
		// Add more currencies as needed
	}

	if symbol, ok := symbols[currency]; ok {
		return symbol
	}
	return currency
}
