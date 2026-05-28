package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// errReader returns a fixed error on Read. Used to exercise the
// body-read-failure path in the webhook handler.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }
func (r errReader) Close() error             { return nil }

// fakeGatewayAdapter implements port.GatewayAdapter and returns a parser
// whose verdict is controlled by the test.
type fakeGatewayAdapter struct {
	parser *fakeWebhookParser
}

func (a *fakeGatewayAdapter) CreateGateway(string) (domain.GatewayProvider, error) {
	return nil, nil
}

func (a *fakeGatewayAdapter) CreateWebhookParser() domain.WebhookParser {
	return a.parser
}

type fakeWebhookParser struct {
	mu          sync.Mutex
	validateErr error
	parsed      domain.PaymentWebhookContext
	parseErr    error
	bodySeen    []byte
}

func (p *fakeWebhookParser) ValidateWebhook(_ context.Context, data []byte, _ string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bodySeen = append([]byte(nil), data...)
	return p.validateErr
}

func (p *fakeWebhookParser) ParseWebhook(_ context.Context, _ []byte) (domain.PaymentWebhookContext, error) {
	if p.parseErr != nil {
		return domain.PaymentWebhookContext{}, p.parseErr
	}
	return p.parsed, nil
}

// newWebhookHandlerForTest wires a real WebhookService against fake repos
// and a stub gateway adapter for the named PSP.
func newWebhookHandlerForTest(
	t *testing.T,
	pspName domain.Gateway,
	parser *fakeWebhookParser,
	idemp *fakeIdempRepo,
	engine *recordingEngine,
) *WebhookHandler {
	t.Helper()
	adapters := map[domain.Gateway]port.GatewayAdapter{pspName: &fakeGatewayAdapter{parser: parser}}
	factory := service.NewGatewayFactory(&fakePspRepo{}, &fakeSettingRepo{}, silentLogger{}, adapters)
	svc := service.NewWebhookService(silentLogger{}, factory, engine, idemp, &fakeSubRepo{})
	return NewWebhookHandler(svc, silentLogger{})
}

func TestWebhookHandler_Process(t *testing.T) {
	t.Run("valid signed body reaches the parser and starts the success workflow", func(t *testing.T) {
		parser := &fakeWebhookParser{parsed: domain.PaymentWebhookContext{
			Type: domain.PaymentSuccess, OrgId: "org_1", OrderId: "ord_1",
		}}
		idemp := &fakeIdempRepo{}
		engine := &recordingEngine{}
		h := newWebhookHandlerForTest(t, "paystack", parser, idemp, engine)

		// No authn — webhooks are unauthenticated; PSP signature is the trust.
		ts := newTestServer()
		h.RegisterRoutes(ts.api())

		body := `{"event":"charge.success","data":{"id":"abc"}}`
		rec := doRaw(t, ts, http.MethodPost, "/api/notify?p=paystack", body)

		require.Equal(t, http.StatusOK, rec.Code, "webhook always returns 200")
		assert.Contains(t, rec.Body.String(), `"success"`)
		assert.Equal(t, []byte(body), parser.bodySeen, "the raw body must reach the parser unchanged")
		assert.Len(t, idemp.created, 1, "idempotency key claimed and retained after success")
		assert.Empty(t, idemp.released, "successful processing must not release the claim")
	})

	t.Run("tampered body — parser rejects validate — webhook still returns 200, claim released for PSP retry", func(t *testing.T) {
		// The endpoint is designed to always return 200 so PSPs don't retry,
		// but a tampered body must NOT proceed to start a workflow. The
		// claim was taken before validation; on failure we release it so a
		// PSP retry (with a correctly-signed body, or after the transient
		// error clears) can run the work.
		parser := &fakeWebhookParser{validateErr: errors.New("bad signature")}
		idemp := &fakeIdempRepo{}
		engine := &recordingEngine{}
		h := newWebhookHandlerForTest(t, "paystack", parser, idemp, engine)

		ts := newTestServer()
		h.RegisterRoutes(ts.api())

		rec := doRaw(t, ts, http.MethodPost, "/api/notify?p=paystack", `{"tampered":true}`)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Len(t, idemp.created, 1, "claim is taken before validation; release reverses it")
		assert.Len(t, idemp.released, 1, "validation failure releases the claim")
	})

	t.Run("unknown psp — no parser available — webhook still returns 200", func(t *testing.T) {
		// The handler swallows the error. Registry doesn't contain the psp.
		idemp := &fakeIdempRepo{}
		engine := &recordingEngine{}
		h := newWebhookHandlerForTest(t, "paystack", &fakeWebhookParser{}, idemp, engine)

		ts := newTestServer()
		h.RegisterRoutes(ts.api())

		rec := doRaw(t, ts, http.MethodPost, "/api/notify?p=unknown", `{"x":1}`)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("body read failure → 400, service is not called", func(t *testing.T) {
		// Previously the handler discarded the io.ReadAll error and proceeded
		// with an empty body, which would surface as an opaque signature
		// failure indistinguishable from a forged request. The handler now
		// surfaces a 400 so the PSP retries instead of recording success.
		parser := &fakeWebhookParser{}
		idemp := &fakeIdempRepo{}
		h := newWebhookHandlerForTest(t, "paystack", parser, idemp, &recordingEngine{})

		ts := newTestServer()
		h.RegisterRoutes(ts.api())

		req := httptest.NewRequest(http.MethodPost, "/api/notify?p=paystack", errReader{err: errors.New("network gone")})
		rec := httptest.NewRecorder()
		asHandler(ts.srv, ts.mws).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Nil(t, parser.bodySeen, "parser must not be invoked when body read fails")
		assert.Empty(t, idemp.created)
	})

	t.Run("idempotency: duplicate webhook is a no-op success", func(t *testing.T) {
		parser := &fakeWebhookParser{parsed: domain.PaymentWebhookContext{Type: domain.PaymentSuccess}}
		idemp := &fakeIdempRepo{exists: true}
		engine := &recordingEngine{}
		h := newWebhookHandlerForTest(t, "paystack", parser, idemp, engine)

		ts := newTestServer()
		h.RegisterRoutes(ts.api())

		rec := doRaw(t, ts, http.MethodPost, "/api/notify?p=paystack", `{"e":1}`)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Nil(t, parser.bodySeen, "parser must not even be invoked when idempotency hits")
	})
}
