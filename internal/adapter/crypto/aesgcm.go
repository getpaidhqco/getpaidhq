package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"getpaidhq/internal/core/port"
)

var _ port.SecretCipher = (*AesGcmCipher)(nil)

// AesGcmCipher implements port.SecretCipher with AES-256-GCM.
//
// Envelope layout (base64-encoded): version byte || 12-byte nonce || ciphertext.
// The version byte selects the key, so rotation is: add a new key under the
// next version, encrypt always uses the newest, decrypt picks by the stored
// byte — old rows stay readable and re-encrypt on their next write.
//
// The AAD is "orgId:id", binding each envelope to the row that owns it: a
// ciphertext copied onto another gateway or org fails authentication instead
// of decrypting in the wrong context.
type AesGcmCipher struct {
	current byte
	keys    map[byte]cipher.AEAD
}

const nonceSize = 12

// NewAesGcmCipher builds the cipher from a base64-encoded 32-byte key
// (env SECRETS_ENCRYPTION_KEY). Today there is a single key at version 1;
// the envelope format already carries the version byte for later rotation.
func NewAesGcmCipher(base64Key string) (*AesGcmCipher, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("SECRETS_ENCRYPTION_KEY is not valid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("SECRETS_ENCRYPTION_KEY must decode to 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCMWithNonceSize(block, nonceSize)
	if err != nil {
		return nil, err
	}
	return &AesGcmCipher{current: 1, keys: map[byte]cipher.AEAD{1: aead}}, nil
}

func aad(orgId, id string) []byte { return []byte(orgId + ":" + id) }

func (c *AesGcmCipher) Encrypt(orgId, id string, plaintext []byte) (string, error) {
	aead := c.keys[c.current]
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nil, nonce, plaintext, aad(orgId, id))
	envelope := make([]byte, 0, 1+nonceSize+len(sealed))
	envelope = append(envelope, c.current)
	envelope = append(envelope, nonce...)
	envelope = append(envelope, sealed...)
	return base64.StdEncoding.EncodeToString(envelope), nil
}

func (c *AesGcmCipher) Decrypt(orgId, id string, envelope string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(envelope)
	if err != nil {
		return nil, errors.New("credentials envelope is not valid base64")
	}
	if len(raw) < 1+nonceSize+1 {
		return nil, errors.New("credentials envelope is truncated")
	}
	aead, ok := c.keys[raw[0]]
	if !ok {
		return nil, fmt.Errorf("credentials envelope has unknown key version %d", raw[0])
	}
	nonce, sealed := raw[1:1+nonceSize], raw[1+nonceSize:]
	plaintext, err := aead.Open(nil, nonce, sealed, aad(orgId, id))
	if err != nil {
		return nil, errors.New("credentials envelope failed authentication")
	}
	return plaintext, nil
}
