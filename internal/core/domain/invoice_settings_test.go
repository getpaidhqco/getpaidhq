package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultInvoiceSettings_FormatReference(t *testing.T) {
	require.Equal(t, "INV-000042", DefaultInvoiceSettings().FormatReference(42))
}

func TestInvoiceSettings_RoundTrip(t *testing.T) {
	cfg := InvoiceSettings{Prefix: "ACME-", Padding: 4}
	raw, err := cfg.Marshal()
	require.NoError(t, err)

	got, err := ParseInvoiceSettings(raw)
	require.NoError(t, err)
	require.Equal(t, cfg, got)
	require.Equal(t, "ACME-0042", got.FormatReference(42))
}

func TestParseInvoiceSettings_EmptyIsDefault(t *testing.T) {
	got, err := ParseInvoiceSettings("")
	require.NoError(t, err)
	require.Equal(t, DefaultInvoiceSettings(), got)
}

func TestParseInvoiceSettings_ZeroFieldsFallBackToDefault(t *testing.T) {
	got, err := ParseInvoiceSettings(`{"prefix":"","padding":0}`)
	require.NoError(t, err)
	require.Equal(t, DefaultInvoiceSettings(), got)
}

func TestParseInvoiceSettings_MalformedErrors(t *testing.T) {
	got, err := ParseInvoiceSettings(`{not valid json`)
	require.Error(t, err)
	require.Equal(t, DefaultInvoiceSettings(), got)
}
