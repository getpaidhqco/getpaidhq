package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// stubGateway is a no-op domain.GatewayProvider.
type stubGateway struct{ domain.GatewayProvider }

// stubWebhookParser is a no-op domain.WebhookParser.
type stubWebhookParser struct{ domain.WebhookParser }

// fakeGatewayAdapter records the settings it was asked to build a gateway from.
type fakeGatewayAdapter struct {
	gateway       domain.GatewayProvider
	createErr     error
	gotSettings   string
	webhookParser domain.WebhookParser
}

func (a *fakeGatewayAdapter) CreateGateway(settingsJSON string) (domain.GatewayProvider, error) {
	a.gotSettings = settingsJSON
	if a.createErr != nil {
		return nil, a.createErr
	}
	return a.gateway, nil
}
func (a *fakeGatewayAdapter) CreateWebhookParser() domain.WebhookParser { return a.webhookParser }

func TestGatewayFactory_NewGateway(t *testing.T) {
	t.Run("looks up psp, settings, and delegates to the registered adapter", func(t *testing.T) {
		adapter := &fakeGatewayAdapter{gateway: stubGateway{}}
		psp := &fakePspRepo{byId: domain.PspConfig{Id: "psp_1", PspId: domain.Paystack}}
		settings := &fakeSettingRepoRW{byId: domain.Setting{Value: `{"secret_key":"sk_test"}`}}
		factory := NewGatewayFactory(psp, settings, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{
			domain.Paystack: adapter,
		})

		got, err := factory.NewGateway(context.Background(), "org_1", "psp_1")

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, `{"secret_key":"sk_test"}`, adapter.gotSettings, "the stored settings value reaches the adapter")
	})

	t.Run("unregistered psp yields an error", func(t *testing.T) {
		psp := &fakePspRepo{byId: domain.PspConfig{Id: "psp_1", PspId: domain.Gateway("UnknownPsp")}}
		settings := &fakeSettingRepoRW{byId: domain.Setting{Value: "{}"}}
		factory := NewGatewayFactory(psp, settings, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{
			domain.Paystack: &fakeGatewayAdapter{},
		})

		_, err := factory.NewGateway(context.Background(), "org_1", "psp_1")
		require.Error(t, err)
	})

	t.Run("psp lookup failure is surfaced", func(t *testing.T) {
		psp := &fakePspRepo{byIdErr: errors.New("missing")}
		factory := NewGatewayFactory(psp, &fakeSettingRepoRW{}, silentLogger{}, nil)

		_, err := factory.NewGateway(context.Background(), "org_1", "psp_x")
		require.Error(t, err)
	})

	t.Run("settings lookup failure is surfaced", func(t *testing.T) {
		psp := &fakePspRepo{byId: domain.PspConfig{Id: "psp_1", PspId: domain.Paystack}}
		settings := &fakeSettingRepoRW{byIdErr: errors.New("missing")}
		factory := NewGatewayFactory(psp, settings, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{domain.Paystack: &fakeGatewayAdapter{}})

		_, err := factory.NewGateway(context.Background(), "org_1", "psp_1")
		require.Error(t, err)
	})

	t.Run("adapter create failure is surfaced", func(t *testing.T) {
		adapter := &fakeGatewayAdapter{createErr: errors.New("bad config")}
		psp := &fakePspRepo{byId: domain.PspConfig{Id: "psp_1", PspId: domain.Paystack}}
		settings := &fakeSettingRepoRW{byId: domain.Setting{Value: "{}"}}
		factory := NewGatewayFactory(psp, settings, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{domain.Paystack: adapter})

		_, err := factory.NewGateway(context.Background(), "org_1", "psp_1")
		require.Error(t, err)
	})
}

func TestGatewayFactory_NewWebhookParser(t *testing.T) {
	t.Run("returns the adapter's parser for a registered psp", func(t *testing.T) {
		adapter := &fakeGatewayAdapter{webhookParser: stubWebhookParser{}}
		factory := NewGatewayFactory(&fakePspRepo{}, &fakeSettingRepoRW{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{
			domain.Paystack: adapter,
		})

		parser := factory.NewWebhookParser(domain.Paystack)
		assert.NotNil(t, parser)
	})

	t.Run("returns nil for an unregistered psp", func(t *testing.T) {
		factory := NewGatewayFactory(&fakePspRepo{}, &fakeSettingRepoRW{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{})

		parser := factory.NewWebhookParser(domain.Gateway("Nope"))
		assert.Nil(t, parser)
	})
}
