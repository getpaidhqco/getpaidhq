package pdf

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/lib"
	"testing"
)

// MockLogger is a mock implementation of the logger.Logger interface
type MockLogger struct{}

func (m MockLogger) Debug(msg string, args ...any) {}
func (m MockLogger) Info(msg string, args ...any)  {}
func (m MockLogger) Warn(msg string, args ...any)  {}
func (m MockLogger) Error(msg string, args ...any) {}
func (m MockLogger) Fatal(msg string, args ...any) {}

func (m MockLogger) Debugf(template string, args ...interface{}) {}
func (m MockLogger) Infof(template string, args ...interface{})  {}
func (m MockLogger) Warnf(template string, args ...interface{})  {}
func (m MockLogger) Errorf(template string, args ...interface{}) {}
func (m MockLogger) Panicf(template string, args ...interface{}) {}
func (m MockLogger) Fatalf(template string, args ...interface{}) {}

func (m MockLogger) Sync() error { return nil }

// TestIntegrationPDFGeneratorGenerate tests the Generate method with actual database lookups
func TestIntegrationPDFGeneratorGenerate(t *testing.T) {
	// Set up database connection
	dbURL := os.Getenv("GETPAIDHQ_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/payloop"
	}

	// Create logger
	log := lib.GetLogger()

	// Create database connection
	db := postgres.NewDatabase(dbURL, log)
	defer db.Close()

	// Create invoice repository
	invoiceRepo := postgres.NewInvoiceRepository(db, log)

	// Create a test context
	ctx := context.Background()

	// Create test data
	orgId := "org_2y05mrf9QE1RbXBDopKyWct3PG1"
	invoiceId := "inv_2ygzzhEtNSby9RRSAmcfCQrEcMe"

	// Create a PDFGenerator with the temp directory as the template directory
	generator := NewPDFGenerator(log)

	// Generate the PDF
	options := GenerateOptions{
		TemplateName: "one.liquid",
	}

	// Fetch the invoice from the database
	dbInvoice, err := invoiceRepo.FindById(ctx, orgId, invoiceId)
	require.NoError(t, err, "Failed to fetch invoice from database")

	// Fetch the line items from the database
	dbLineItems, err := invoiceRepo.ListLineItems(ctx, orgId, invoiceId)
	require.NoError(t, err, "Failed to fetch line items from database")

	// Generate the PDF using the data from the database
	// Attach line items to invoice for PDF generation
	dbInvoice.LineItems = dbLineItems
	pdfBytes, err := generator.Generate(dbInvoice, options)

	// Verify the result
	assert.NoError(t, err, "Failed to generate PDF")
	assert.NotEmpty(t, pdfBytes, "PDF content should not be empty")

	// Optionally, write the PDF to a file for manual inspection
	outputPath := filepath.Join("./", "output.pdf")
	err = os.WriteFile(outputPath, pdfBytes, 0644)
	require.NoError(t, err, "Failed to write PDF to file")
	t.Logf("PDF generated at %s", outputPath)
}
