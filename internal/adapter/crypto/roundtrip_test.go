package crypto

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// memPspRepo is a single-row in-memory port.PspRepository.
type memPspRepo struct{ row domain.PspConfig }

func (r *memPspRepo) Create(_ context.Context, c domain.PspConfig) (domain.PspConfig, error) {
	r.row = c
	return c, nil
}
func (r *memPspRepo) FindById(_ context.Context, _, _ string) (domain.PspConfig, error) {
	return r.row, nil
}

// recordingAdapter captures what the factory hands the gateway adapter.
type recordingAdapter struct {
	gotConfig map[string]string
	gotCreds  map[string]domain.Secret
}

func (a *recordingAdapter) CreateGateway(config map[string]string, creds map[string]domain.Secret) (domain.GatewayProvider, error) {
	a.gotConfig, a.gotCreds = config, creds
	return nil, nil
}
func (a *recordingAdapter) CreateWebhookParser() domain.WebhookParser { return nil }

// noopLogger / noopPubSub satisfy the ports by embedding the interface and
// overriding nothing the roundtrip exercises.
type noopLogger struct{ port.Logger }

func (noopLogger) Errorf(string, ...any) {}

type noopPubSub struct{ port.PubSub }

func (noopPubSub) Publish(string, string, any) error { return nil }

// TestServiceRoundTripWithRealCipher seals credentials through
// PspService.CreateGateway and opens them through GatewayFactory.NewGateway
// using the real AES-GCM cipher — the exact production path minus postgres.
func TestServiceRoundTripWithRealCipher(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	cipher, err := NewAesGcmCipher(base64.StdEncoding.EncodeToString(key))
	require.NoError(t, err)

	repo := &memPspRepo{}
	svc := service.NewPspService(repo, cipher, noopLogger{}, noopPubSub{})

	created, err := svc.CreateGateway(context.Background(), port.CreateGatewayInput{
		OrgId: "org_1", PspId: domain.Paystack, Name: "Primary",
		Config:      map[string]string{"connect_id": "cn_1"},
		Credentials: map[string]domain.Secret{"api_key": "sk_live_roundtrip"},
	})
	require.NoError(t, err)
	assert.NotContains(t, repo.row.EncryptedCredentials, "sk_live_roundtrip", "stored envelope is ciphertext")

	adapter := &recordingAdapter{}
	factory := service.NewGatewayFactory(repo, cipher, noopLogger{}, map[domain.Gateway]port.GatewayAdapter{
		domain.Paystack: adapter,
	})

	_, err = factory.NewGateway(context.Background(), "org_1", created.Id)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"connect_id": "cn_1"}, adapter.gotConfig)
	assert.Equal(t, "sk_live_roundtrip", adapter.gotCreds["api_key"].Reveal())
}
