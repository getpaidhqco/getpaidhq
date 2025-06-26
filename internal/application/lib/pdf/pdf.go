package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
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

// ChromedpEngine is an implementation of PDFEngine using chromedp
type ChromedpEngine struct {
	timeout time.Duration
}

// NewChromedpEngine creates a new ChromedpEngine with default settings
func NewChromedpEngine() *ChromedpEngine {
	return &ChromedpEngine{
		timeout: 30 * time.Second,
	}
}

// Generate generates a PDF from HTML content using chromedp
func (e ChromedpEngine) Generate(htmlContent string) ([]byte, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// Set up chromedp options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("disable-default-apps", true),
	)

	// Create context
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	// Generate PDF
	var pdfBytes []byte

	// Create a temporary HTML file
	tempFile, err := os.CreateTemp("", "pdf-*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath) // Clean up the temp file when done

	// Write HTML content to the temp file
	if _, err := tempFile.WriteString(htmlContent); err != nil {
		return nil, fmt.Errorf("failed to write HTML to temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Convert the file path to a URL
	fileURL := fmt.Sprintf("file://%s", tempFilePath)

	err = chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				WithPaperWidth(8.27).
				WithPaperHeight(11.7).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBytes = buf
			return nil
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return pdfBytes, nil
}

// PDFGenerator is a utility for generating PDF documents from templates
type PDFGenerator struct {
	logger         logger.Logger
	templateEngine TemplateEngine
	fileSystem     FileSystem
	pdfEngine      PDFEngine
	templateDir    string
}

// NewPDFGenerator creates a new PDFGenerator instance with default dependencies
func NewPDFGenerator(logger logger.Logger) *PDFGenerator {
	templateEngine := NewLiquidTemplateEngine()
	return &PDFGenerator{
		logger:         logger,
		templateEngine: templateEngine,
		fileSystem:     DefaultFileSystem{},
		pdfEngine:      NewChromedpEngine(),
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

// Generate generates a PDF from an invoice using the specified template (new aggregate version)
func (g *PDFGenerator) Generate(invoice entities.Invoice, options GenerateOptions) ([]byte, error) {
	return g.GenerateWithLineItems(invoice, invoice.LineItems, options)
}

// GenerateWithLineItems generates a PDF from an invoice using the specified template (backwards compatible)
func (g *PDFGenerator) GenerateWithLineItems(invoice entities.Invoice, lineItems []entities.InvoiceLineItem, options GenerateOptions) ([]byte, error) {
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

	outputPath := options.OutputPath
	if outputPath == "" {
		tmpFile, err := os.CreateTemp("", "invoice-*.pdf")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		defer tmpFile.Close()
		outputPath = tmpFile.Name()
	}
	err = g.fileSystem.WriteFile(outputPath, pdfBytes, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write PDF to file: %w", err)
	}

	if g.logger != nil {
		g.logger.Debugf("PDF generated at %s", outputPath)
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
