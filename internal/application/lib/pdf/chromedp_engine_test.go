package pdf

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestChromedpEngineGenerate(t *testing.T) {
	// Skip this test in CI environments
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a new ChromedpEngine
	engine := NewChromedpEngine()

	// Simple HTML content
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <title>Test PDF</title>
</head>
<body>
    <h1>Test PDF Generation</h1>
    <p>This is a test of the PDF generation functionality.</p>
    <p>If this works, the PDF should not be empty.</p>
</body>
</html>
`

	// Generate PDF
	pdfBytes, err := engine.Generate(htmlContent)

	// Verify the result
	assert.NoError(t, err, "Failed to generate PDF")
	assert.NotEmpty(t, pdfBytes, "PDF content should not be empty")
	assert.Greater(t, len(pdfBytes), 100, "PDF content should be substantial")
}