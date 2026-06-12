package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testKey(t *testing.T) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString(make([]byte, 32))
}

func TestNewAesGcmCipher(t *testing.T) {
	t.Run("rejects non-base64 key", func(t *testing.T) {
		_, err := NewAesGcmCipher("not base64!!!")
		require.Error(t, err)
	})

	t.Run("rejects wrong-length key", func(t *testing.T) {
		_, err := NewAesGcmCipher(base64.StdEncoding.EncodeToString(make([]byte, 16)))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})
}

func TestAesGcmCipher_RoundTrip(t *testing.T) {
	c, err := NewAesGcmCipher(testKey(t))
	require.NoError(t, err)

	plaintext := []byte(`{"secret_key":"sk_live_abc"}`)
	envelope, err := c.Encrypt("org_1", "psp_1", plaintext)
	require.NoError(t, err)
	assert.NotContains(t, envelope, "sk_live_abc")

	got, err := c.Decrypt("org_1", "psp_1", envelope)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestAesGcmCipher_EnvelopesAreNonDeterministic(t *testing.T) {
	c, err := NewAesGcmCipher(testKey(t))
	require.NoError(t, err)

	a, err := c.Encrypt("org_1", "psp_1", []byte("same"))
	require.NoError(t, err)
	b, err := c.Encrypt("org_1", "psp_1", []byte("same"))
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "random nonce must make envelopes differ")
}

func TestAesGcmCipher_AadBindsEnvelopeToRow(t *testing.T) {
	c, err := NewAesGcmCipher(testKey(t))
	require.NoError(t, err)

	envelope, err := c.Encrypt("org_1", "psp_1", []byte("secret"))
	require.NoError(t, err)

	t.Run("other org fails", func(t *testing.T) {
		_, err := c.Decrypt("org_2", "psp_1", envelope)
		require.Error(t, err)
	})
	t.Run("other id fails", func(t *testing.T) {
		_, err := c.Decrypt("org_1", "psp_2", envelope)
		require.Error(t, err)
	})
}

func TestAesGcmCipher_RejectsBadEnvelopes(t *testing.T) {
	c, err := NewAesGcmCipher(testKey(t))
	require.NoError(t, err)

	envelope, err := c.Encrypt("org_1", "psp_1", []byte("secret"))
	require.NoError(t, err)

	t.Run("tampered ciphertext fails authentication", func(t *testing.T) {
		raw, _ := base64.StdEncoding.DecodeString(envelope)
		raw[len(raw)-1] ^= 0xFF
		_, err := c.Decrypt("org_1", "psp_1", base64.StdEncoding.EncodeToString(raw))
		require.Error(t, err)
	})

	t.Run("unknown key version", func(t *testing.T) {
		raw, _ := base64.StdEncoding.DecodeString(envelope)
		raw[0] = 9
		_, err := c.Decrypt("org_1", "psp_1", base64.StdEncoding.EncodeToString(raw))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "version")
	})

	t.Run("not base64", func(t *testing.T) {
		_, err := c.Decrypt("org_1", "psp_1", "%%%")
		require.Error(t, err)
	})

	t.Run("truncated", func(t *testing.T) {
		_, err := c.Decrypt("org_1", "psp_1", base64.StdEncoding.EncodeToString([]byte{1, 2, 3}))
		require.Error(t, err)
	})
}
