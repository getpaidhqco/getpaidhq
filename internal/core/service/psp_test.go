package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// fakePspRepo records PSP config creation and serves a configurable lookup.
type fakePspRepo struct {
	port.PspRepository
	byId      domain.PspConfig
	byIdErr   error
	createErr error
	created   []domain.PspConfig
}

func (r *fakePspRepo) Create(_ context.Context, c domain.PspConfig) (domain.PspConfig, error) {
	if r.createErr != nil {
		return domain.PspConfig{}, r.createErr
	}
	r.created = append(r.created, c)
	return c, nil
}

func (r *fakePspRepo) FindById(_ context.Context, _, _ string) (domain.PspConfig, error) {
	if r.byIdErr != nil {
		return domain.PspConfig{}, r.byIdErr
	}
	return r.byId, nil
}

// fakeSecretCipher is a reversible stand-in for the AES-GCM cipher that still
// enforces AAD binding: an envelope only opens with the (orgId, id) it was
// sealed for. Envelope shape: "enc[orgId:id]" + plaintext.
type fakeSecretCipher struct {
	encryptErr error
	decryptErr error
}

func (c *fakeSecretCipher) Encrypt(orgId, id string, plaintext []byte) (string, error) {
	if c.encryptErr != nil {
		return "", c.encryptErr
	}
	return "enc[" + orgId + ":" + id + "]" + string(plaintext), nil
}

func (c *fakeSecretCipher) Decrypt(orgId, id string, envelope string) ([]byte, error) {
	if c.decryptErr != nil {
		return nil, c.decryptErr
	}
	prefix := "enc[" + orgId + ":" + id + "]"
	if !strings.HasPrefix(envelope, prefix) {
		return nil, errors.New("envelope failed authentication")
	}
	return []byte(strings.TrimPrefix(envelope, prefix)), nil
}

func TestPspService_CreateGateway(t *testing.T) {
	t.Run("stores config readable and credentials sealed", func(t *testing.T) {
		psp := &fakePspRepo{}
		cipher := &fakeSecretCipher{}
		svc := NewPspService(psp, cipher, silentLogger{}, &recordingPubSub{})

		got, err := svc.CreateGateway(context.Background(), port.CreateGatewayInput{
			OrgId: "org_1", PspId: domain.Paystack, Name: "Primary",
			Config:      map[string]string{"connect_id": "cn_1"},
			Credentials: map[string]domain.Secret{"api_key": "sk_test"},
		})

		require.NoError(t, err)
		assert.True(t, got.Active, "newly created gateway is active")
		assert.Equal(t, domain.Paystack, got.PspId)
		require.Len(t, psp.created, 1)
		row := psp.created[0]
		assert.Equal(t, map[string]string{"connect_id": "cn_1"}, row.Config)
		// The stored envelope is sealed for this org+gateway (real ciphertext
		// opacity is covered by the adapter/crypto tests; the fake is reversible).
		assert.True(t, strings.HasPrefix(row.EncryptedCredentials, "enc[org_1:"+row.Id+"]"))
		opened, err := cipher.Decrypt("org_1", row.Id, row.EncryptedCredentials)
		require.NoError(t, err)
		var creds map[string]string
		require.NoError(t, json.Unmarshal(opened, &creds))
		assert.Equal(t, map[string]string{"api_key": "sk_test"}, creds)
	})

	t.Run("missing credentials are rejected", func(t *testing.T) {
		psp := &fakePspRepo{}
		svc := NewPspService(psp, &fakeSecretCipher{}, silentLogger{}, &recordingPubSub{})

		_, err := svc.CreateGateway(context.Background(), port.CreateGatewayInput{OrgId: "org_1", PspId: domain.Paystack})

		require.Error(t, err)
		assert.Empty(t, psp.created)
	})

	t.Run("nil cipher (no SECRETS_ENCRYPTION_KEY) is a clear error", func(t *testing.T) {
		psp := &fakePspRepo{}
		svc := NewPspService(psp, nil, silentLogger{}, &recordingPubSub{})

		_, err := svc.CreateGateway(context.Background(), port.CreateGatewayInput{
			OrgId: "org_1", PspId: domain.Paystack,
			Credentials: map[string]domain.Secret{"api_key": "sk_test"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "SECRETS_ENCRYPTION_KEY")
		assert.Empty(t, psp.created, "nothing stored without a cipher")
	})

	t.Run("encrypt failure short-circuits before the row is written", func(t *testing.T) {
		psp := &fakePspRepo{}
		svc := NewPspService(psp, &fakeSecretCipher{encryptErr: errors.New("hsm down")}, silentLogger{}, &recordingPubSub{})

		_, err := svc.CreateGateway(context.Background(), port.CreateGatewayInput{
			OrgId: "org_1", PspId: domain.Paystack,
			Credentials: map[string]domain.Secret{"api_key": "sk_test"},
		})

		require.Error(t, err)
		assert.Empty(t, psp.created)
	})

	t.Run("psp create failure is surfaced", func(t *testing.T) {
		psp := &fakePspRepo{createErr: errors.New("db down")}
		svc := NewPspService(psp, &fakeSecretCipher{}, silentLogger{}, &recordingPubSub{})

		_, err := svc.CreateGateway(context.Background(), port.CreateGatewayInput{
			OrgId: "org_1", PspId: domain.Paystack,
			Credentials: map[string]domain.Secret{"api_key": "sk_test"},
		})

		require.Error(t, err)
	})
}
