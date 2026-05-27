package cognito

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE on testability: the public constructor NewCognitoClient performs a live
// HTTP GET of the Cognito JWKS endpoint, and NewCognitoMiddleware wraps it (and
// panics on failure), so neither can be unit-tested without network access /
// live infra. The signature-verification + claim-mapping logic, however, is
// pure given a key set. These tests build a Cognito client backed by a locally
// generated RSA key, mint self-signed tokens, and exercise VerifyToken and the
// CognitoMiddleware.Authenticate claim->AuthUser mapping end to end offline.

const (
	testKid      = "test-kid"
	testClientId = "test-client-id"
	testIss      = "https://cognito-idp.test.amazonaws.com/pool"
)

type cognitoTestHarness struct {
	key    *rsa.PrivateKey
	client Cognito
}

func newHarness(t *testing.T) cognitoTestHarness {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	client := Cognito{
		ClientId: testClientId,
		Iss:      testIss,
		PublicKeys: PublicKeys{
			testKid: PublicKey{Kid: testKid, Kty: "RSA", PEM: &key.PublicKey},
		},
	}
	return cognitoTestHarness{key: key, client: client}
}

// signToken mints an RS256 JWT signed with the harness key, with kid set so
// getCert can find the matching public key.
func (h cognitoTestHarness) signToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKid
	signed, err := tok.SignedString(h.key)
	require.NoError(t, err)
	return signed
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"aud":            testClientId,
		"iss":            testIss,
		"exp":            time.Now().Add(time.Hour).Unix(),
		"sub":            "user-123",
		"email":          "user@example.com",
		"custom:company": "org-789",
		"cognito:groups": []any{"Admin", "Support"},
	}
}

func TestCognito_VerifyToken(t *testing.T) {
	h := newHarness(t)

	t.Run("valid token verifies", func(t *testing.T) {
		tok := h.signToken(t, validClaims())
		parsed, err := h.client.VerifyToken(tok)
		require.NoError(t, err)
		assert.True(t, parsed.Valid)
	})

	t.Run("wrong audience is rejected", func(t *testing.T) {
		c := validClaims()
		c["aud"] = "some-other-client"
		_, err := h.client.VerifyToken(h.signToken(t, c))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audience is invalid")
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		c := validClaims()
		c["exp"] = time.Now().Add(-time.Hour).Unix()
		_, err := h.client.VerifyToken(h.signToken(t, c))
		require.Error(t, err)
	})

	t.Run("wrong issuer is rejected", func(t *testing.T) {
		c := validClaims()
		c["iss"] = "https://evil.example.com"
		_, err := h.client.VerifyToken(h.signToken(t, c))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "iss is invalid")
	})

	t.Run("unknown kid is rejected", func(t *testing.T) {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims())
		tok.Header["kid"] = "unknown-kid"
		signed, err := tok.SignedString(h.key)
		require.NoError(t, err)
		_, err = h.client.VerifyToken(signed)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid kid")
	})

	t.Run("non-RS256 signing method is rejected", func(t *testing.T) {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims())
		tok.Header["kid"] = testKid
		signed, err := tok.SignedString([]byte("hmac-secret"))
		require.NoError(t, err)
		_, err = h.client.VerifyToken(signed)
		require.Error(t, err)
	})

	t.Run("signature from a different key is rejected", func(t *testing.T) {
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims())
		tok.Header["kid"] = testKid
		signed, err := tok.SignedString(otherKey)
		require.NoError(t, err)
		_, err = h.client.VerifyToken(signed)
		require.Error(t, err)
	})
}

func TestCognitoMiddleware_Authenticate(t *testing.T) {
	h := newHarness(t)
	mw := CognitoMiddleware{logger: noopLogger{}, env: lib.Env{}, client: h.client}

	t.Run("valid token maps claims to AuthUser with lowercased roles", func(t *testing.T) {
		user, err := mw.Authenticate(t.Context(), h.signToken(t, validClaims()))
		require.NoError(t, err)
		assert.Equal(t, "org-789", user.OrgId)
		assert.Equal(t, "user-123", user.Id)
		assert.Equal(t, "user@example.com", user.Email)
		assert.Equal(t, []port.UserRole{port.RoleAdmin, port.RoleSupport}, user.Roles)
		// PrimaryRole is derived: admin outranks support.
		assert.Equal(t, port.RoleAdmin, user.PrimaryRole)
	})

	t.Run("invalid token is rejected", func(t *testing.T) {
		c := validClaims()
		c["exp"] = time.Now().Add(-time.Hour).Unix()
		_, err := mw.Authenticate(t.Context(), h.signToken(t, c))
		require.Error(t, err)
	})

	t.Run("empty company claim is rejected as invalid token", func(t *testing.T) {
		c := validClaims()
		c["custom:company"] = ""
		_, err := mw.Authenticate(t.Context(), h.signToken(t, c))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token")
	})
}

func TestNewCognitoClient_Validation(t *testing.T) {
	t.Run("missing region and pool id returns ErrInvalidParam without network call", func(t *testing.T) {
		_, err := NewCognitoClient(lib.Env{CognitoClientId: "x"})
		require.ErrorIs(t, err, ErrInvalidParam)
	})
}

func TestParsePEM(t *testing.T) {
	tests := []struct {
		name    string
		key     PublicKey
		wantErr string
	}{
		{
			name:    "non-RSA kty rejected",
			key:     PublicKey{Kty: "EC", E: "AQAB", N: "AQ"},
			wantErr: "must be RSA",
		},
		{
			name:    "unsupported exponent rejected",
			key:     PublicKey{Kty: "RSA", E: "BADEXP", N: "AQ"},
			wantErr: "is invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePEM(tt.key)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}

	t.Run("valid RSA key parses to a 65537 exponent", func(t *testing.T) {
		// base64url "AQAB" is the standard 65537 exponent; N must be valid base64url.
		pk, err := parsePEM(PublicKey{Kty: "RSA", E: "AQAB", N: "sXch"})
		require.NoError(t, err)
		assert.Equal(t, 65537, pk.E)
	})
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...any)  {}
func (noopLogger) Info(string, ...any)   {}
func (noopLogger) Warn(string, ...any)   {}
func (noopLogger) Error(string, ...any)  {}
func (noopLogger) Fatal(string, ...any)  {}
func (noopLogger) Debugf(string, ...any) {}
func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Warnf(string, ...any)  {}
func (noopLogger) Errorf(string, ...any) {}
func (noopLogger) Panicf(string, ...any) {}
func (noopLogger) Fatalf(string, ...any) {}
func (noopLogger) Sync() error           { return nil }
