package service

import (
	"context"
	"testing"

	"getpaidhq/internal/core/domain"

	"github.com/stretchr/testify/require"
)

func TestInvoiceSettingsService_Resolve_DefaultWhenMissing(t *testing.T) {
	repo := newMapSettingRepo()
	svc := NewInvoiceSettingsService(repo, silentLogger{})

	cfg, err := svc.ResolveInvoiceSettings(context.Background(), "org_x")
	require.NoError(t, err)
	require.Equal(t, domain.DefaultInvoiceSettings(), cfg)
}

func TestInvoiceSettingsService_SetThenResolve(t *testing.T) {
	repo := newMapSettingRepo()
	svc := NewInvoiceSettingsService(repo, silentLogger{})

	want := domain.InvoiceSettings{Prefix: "ACME-", Padding: 4}
	require.NoError(t, svc.SetInvoiceSettings(context.Background(), "org_x", want))

	got, err := svc.ResolveInvoiceSettings(context.Background(), "org_x")
	require.NoError(t, err)
	require.Equal(t, want, got)
}
