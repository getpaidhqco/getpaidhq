package checkout_com

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/port"
)

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
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(body))
	return hex.EncodeToString(h.Sum(nil))
}

func TestValidateWebhook_AcceptsCorrectSignature(t *testing.T) {
	p := NewWebhookParser(noopLogger{}, "secret_xyz")
	body := []byte(`{"type":"payment_approved"}`)

	err := p.ValidateWebhook(context.Background(), body, sign("secret_xyz", string(body)))
	require.NoError(t, err)
}

func TestValidateWebhook_RejectsForgedSignature(t *testing.T) {
	p := NewWebhookParser(noopLogger{}, "secret_xyz")
	body := []byte(`{"type":"payment_approved"}`)

	err := p.ValidateWebhook(context.Background(), body, sign("wrong_secret", string(body)))
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestValidateWebhook_RejectsMissingSignature(t *testing.T) {
	p := NewWebhookParser(noopLogger{}, "secret_xyz")
	err := p.ValidateWebhook(context.Background(), []byte(`{}`), "")
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestValidateWebhook_FailsClosedWithoutSecret(t *testing.T) {
	p := NewWebhookParser(noopLogger{}, "")
	err := p.ValidateWebhook(context.Background(), []byte(`{}`), "anything")
	assert.ErrorIs(t, err, ErrMissingWebhookSecret)
}
