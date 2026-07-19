package apikey

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

// ErrMissingApiKeyPepper is returned by Hash when no pepper is
// configured. Without a pepper the hash is just SHA-256 of the key,
// which is rainbow-table-cheap for short keys — fail closed.
var ErrMissingApiKeyPepper = errors.New("API_KEY_PEPPER not set: api key hashing requires a server-side pepper")

// GenerateSecret returns 32 bytes of crypto/rand-sourced
// randomness, base64-URL-encoded. Caller MUST surface this exactly
// once to the user (it's their copy of the key); we never store it.
func GenerateSecret() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// Hash computes HMAC-SHA256(pepper, rawKey) and hex-encodes the
// result. Both the create path and the verify path call this; the
// hash is the value that lives in the DB.
//
// The pepper is intentionally a server-secret, NOT a per-row salt:
// salts protect against rainbow tables on a stolen DB, peppers
// protect against DB-only compromise. For API keys you want both —
// but the keys themselves are 256 random bits, so rainbow tables are
// already infeasible and a pepper is the cheaper safeguard.
func Hash(rawKey, pepper string) (string, error) {
	if pepper == "" {
		return "", ErrMissingApiKeyPepper
	}
	mac := hmac.New(sha256.New, []byte(pepper))
	mac.Write([]byte(rawKey))
	return hex.EncodeToString(mac.Sum(nil)), nil
}
