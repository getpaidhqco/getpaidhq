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

// stubGateway is a no-op port.PaymentGateway.
type stubGateway struct{ port.PaymentGateway }

// stubWebhookParser is a no-op domain.WebhookParser.
type stubWebhookParser struct{ domain.WebhookParser }

// fakeGatewayAdapter records the config/credentials it was asked to build a
// gateway from.
type fakeGatewayAdapter struct {
	gateway       port.PaymentGateway
	createErr     error
	gotConfig     map[string]string
	gotCreds      map[string]domain.Secret
	webhookParser domain.WebhookParser
}

func (a *fakeGatewayAdapter) CreateGateway(config map[string]string, credentials map[string]domain.Secret) (port.PaymentGateway, error) {
	a.gotConfig = config
	a.gotCreds = credentials
	if a.createErr != nil {
		return nil, a.createErr
	}
	return a.gateway, nil
}
func (a *fakeGatewayAdapter) CreateWebhookParser() domain.WebhookParser { return a.webhookParser }

func TestGatewayFactory_NewGateway(t *testing.T) {
	cipher := &fakeSecretCipher{}
	sealed, _ := cipher.Encrypt("org_1", "psp_1", []byte(`{"api_key":"sk_test"}`))

	t.Run("looks up the psp row, opens credentials, and delegates to the registered adapter", func(t *testing.T) {
		adapter := &fakeGatewayAdapter{gateway: stubGateway{}}
		psp := &fakePspRepo{byId: domain.PspConfig{
			OrgId: "org_1", Id: "psp_1", PspId: domain.Paystack,
			Config:               map[string]string{"connect_id": "cn_1"},
			EncryptedCredentials: sealed,
		}}
		factory := NewGatewayFactory(psp, cipher, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{
			domain.Paystack: adapter,
		})

		got, err := factory.NewGateway(context.Background(), "org_1", "psp_1")

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, map[string]string{"connect_id": "cn_1"}, adapter.gotConfig)
		require.Contains(t, adapter.gotCreds, "api_key")
		assert.Equal(t, "sk_test", adapter.gotCreds["api_key"].Reveal(), "the adapter receives the opened secret")
	})

	t.Run("no stored credentials yields an empty map, not an error", func(t *testing.T) {
		adapter := &fakeGatewayAdapter{gateway: stubGateway{}}
		psp := &fakePspRepo{byId: domain.PspConfig{OrgId: "org_1", Id: "psp_1", PspId: domain.Memory}}
		factory := NewGatewayFactory(psp, cipher, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{
			domain.Memory: adapter,
		})

		_, err := factory.NewGateway(context.Background(), "org_1", "psp_1")

		require.NoError(t, err)
		assert.Empty(t, adapter.gotCreds)
	})

	t.Run("unregistered psp yields an error", func(t *testing.T) {
		psp := &fakePspRepo{byId: domain.PspConfig{OrgId: "org_1", Id: "psp_1", PspId: domain.Gateway("UnknownPsp")}}
		factory := NewGatewayFactory(psp, cipher, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{
			domain.Paystack: &fakeGatewayAdapter{},
		})

		_, err := factory.NewGateway(context.Background(), "org_1", "psp_1")
		require.Error(t, err)
	})

	t.Run("psp lookup failure is surfaced", func(t *testing.T) {
		psp := &fakePspRepo{byIdErr: errors.New("missing")}
		factory := NewGatewayFactory(psp, cipher, silentLogger{}, nil)

		_, err := factory.NewGateway(context.Background(), "org_1", "psp_x")
		require.Error(t, err)
	})

	t.Run("decrypt failure is surfaced and the adapter is never reached", func(t *testing.T) {
		adapter := &fakeGatewayAdapter{gateway: stubGateway{}}
		psp := &fakePspRepo{byId: domain.PspConfig{
			OrgId: "org_2", Id: "psp_1", PspId: domain.Paystack,
			EncryptedCredentials: sealed, // sealed for org_1 — fails AAD for org_2
		}}
		factory := NewGatewayFactory(psp, cipher, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{domain.Paystack: adapter})

		_, err := factory.NewGateway(context.Background(), "org_2", "psp_1")

		require.Error(t, err)
		assert.Nil(t, adapter.gotCreds, "adapter never sees a failed decrypt")
	})

	t.Run("nil cipher with stored credentials is a clear error", func(t *testing.T) {
		psp := &fakePspRepo{byId: domain.PspConfig{
			OrgId: "org_1", Id: "psp_1", PspId: domain.Paystack,
			EncryptedCredentials: sealed,
		}}
		factory := NewGatewayFactory(psp, nil, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{domain.Paystack: &fakeGatewayAdapter{}})

		_, err := factory.NewGateway(context.Background(), "org_1", "psp_1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SECRETS_ENCRYPTION_KEY")
	})

	t.Run("adapter create failure is surfaced", func(t *testing.T) {
		adapter := &fakeGatewayAdapter{createErr: errors.New("bad config")}
		psp := &fakePspRepo{byId: domain.PspConfig{OrgId: "org_1", Id: "psp_1", PspId: domain.Paystack, EncryptedCredentials: sealed}}
		factory := NewGatewayFactory(psp, cipher, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{domain.Paystack: adapter})

		_, err := factory.NewGateway(context.Background(), "org_1", "psp_1")
		require.Error(t, err)
	})
}

func TestGatewayFactory_NewWebhookParser(t *testing.T) {
	t.Run("returns the adapter's parser for a registered psp", func(t *testing.T) {
		adapter := &fakeGatewayAdapter{webhookParser: stubWebhookParser{}}
		factory := NewGatewayFactory(&fakePspRepo{}, &fakeSecretCipher{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{
			domain.Paystack: adapter,
		})

		parser := factory.NewWebhookParser(domain.Paystack)
		assert.NotNil(t, parser)
	})

	t.Run("returns nil for an unregistered psp", func(t *testing.T) {
		factory := NewGatewayFactory(&fakePspRepo{}, &fakeSecretCipher{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{})

		parser := factory.NewWebhookParser(domain.Gateway("Nope"))
		assert.Nil(t, parser)
	})
}
