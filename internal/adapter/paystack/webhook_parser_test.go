package paystack

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/port"
)

// noopLogger satisfies port.Logger silently for tests.
type noopLogger struct{ port.Logger }

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

func sign(secret, body string) string {
	h := hmac.New(sha512.New, []byte(secret))
	h.Write([]byte(body))
	return hex.EncodeToString(h.Sum(nil))
}

func TestValidateWebhook_AcceptsCorrectSignature(t *testing.T) {
	p := NewWebhookParser(nil, PaystackFactory{}, noopLogger{}, "secret_xyz")
	body := []byte(`{"event":"charge.success"}`)

	err := p.ValidateWebhook(context.Background(), body, sign("secret_xyz", string(body)))
	require.NoError(t, err)
}

func TestValidateWebhook_RejectsForgedSignature(t *testing.T) {
	p := NewWebhookParser(nil, PaystackFactory{}, noopLogger{}, "secret_xyz")
	body := []byte(`{"event":"charge.success"}`)

	err := p.ValidateWebhook(context.Background(), body, sign("not_the_real_secret", string(body)))
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestValidateWebhook_RejectsMissingSignature(t *testing.T) {
	p := NewWebhookParser(nil, PaystackFactory{}, noopLogger{}, "secret_xyz")
	body := []byte(`{"event":"charge.success"}`)

	err := p.ValidateWebhook(context.Background(), body, "")
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestValidateWebhook_RejectsTamperedBody(t *testing.T) {
	p := NewWebhookParser(nil, PaystackFactory{}, noopLogger{}, "secret_xyz")
	original := `{"event":"charge.success","amount":100}`
	tampered := `{"event":"charge.success","amount":9999}`

	// Caller signs the tampered body with the right secret — but the
	// real Paystack would have signed the original; we feed the
	// original signature against the tampered body to simulate a
	// payload-modification attack.
	originalSig := sign("secret_xyz", original)
	err := p.ValidateWebhook(context.Background(), []byte(tampered), originalSig)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestValidateWebhook_FailsClosedWithoutSecret(t *testing.T) {
	p := NewWebhookParser(nil, PaystackFactory{}, noopLogger{}, "")
	body := []byte(`{}`)

	err := p.ValidateWebhook(context.Background(), body, "anything")
	assert.ErrorIs(t, err, ErrMissingWebhookSecret)
}
