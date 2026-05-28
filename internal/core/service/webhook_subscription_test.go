package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type fakeWebhookSubRepo struct {
	port.WebhookSubscriptionRepository
	createErr error
	created   []domain.WebhookSubscription
}

func (r *fakeWebhookSubRepo) Create(_ context.Context, w domain.WebhookSubscription) (domain.WebhookSubscription, error) {
	if r.createErr != nil {
		return domain.WebhookSubscription{}, r.createErr
	}
	r.created = append(r.created, w)
	return w, nil
}

// newWebhookSubscriptionService builds the service with the SSRF
// predicate relaxed to allow loopback — required for httptest.Server,
// which always binds to 127.0.0.1. Never relax the predicate in prod
// wiring (config.NewApp); the package-private field is the only path.
func newWebhookSubscriptionService(repo port.WebhookSubscriptionRepository, ps port.PubSub) *WebhookSubscriptionService {
	if ps == nil {
		ps = &recordingPubSub{}
	}
	svc := NewWebhookSubscriptionService(silentLogger{}, repo, nil, ps)
	svc.ipPredicate = allowAllIPs
	return svc
}

func TestWebhookSubscriptionService_Create(t *testing.T) {
	t.Run("persists and publishes webhook.created", func(t *testing.T) {
		repo := &fakeWebhookSubRepo{}
		ps := &recordingPubSub{}
		svc := newWebhookSubscriptionService(repo, ps)

		got, err := svc.Create(context.Background(), domain.CreateWebhookSubscriptionInput{
			OrgId: "org_1", Url: "https://example.com/hook", Events: []string{"order.completed"}, Secret: "sek",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, got.Id)
		assert.Equal(t, "org_1", got.OrgID)
		require.Len(t, repo.created, 1)
		assert.True(t, ps.hasTopic(port.TopicWebhookSubscriptionCreated))
	})

	t.Run("repo failure surfaces and does not publish", func(t *testing.T) {
		repo := &fakeWebhookSubRepo{createErr: errors.New("db down")}
		ps := &recordingPubSub{}
		svc := newWebhookSubscriptionService(repo, ps)

		_, err := svc.Create(context.Background(), domain.CreateWebhookSubscriptionInput{OrgId: "org_1"})

		require.Error(t, err)
		assert.False(t, ps.hasTopic(port.TopicWebhookSubscriptionCreated))
	})
}

func TestWebhookSubscriptionService_SendWebhook(t *testing.T) {
	t.Run("POSTs the event and signs it when a secret is set", func(t *testing.T) {
		var (
			mu        sync.Mutex
			gotSig    string
			gotBody   []byte
			gotMethod string
		)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			gotMethod = r.Method
			gotSig = r.Header.Get("X-Signature")
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		svc := newWebhookSubscriptionService(&fakeWebhookSubRepo{}, nil)

		err := svc.SendWebhook(context.Background(), port.OutgoingWebhookPayload{
			WebhookSubscription: domain.WebhookSubscription{URL: srv.URL, Secret: "sek"},
			Event:               port.PubSubPayload{Topic: "order.completed", OrgId: "org_1"},
		})

		require.NoError(t, err)
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, http.MethodPost, gotMethod)
		assert.NotEmpty(t, gotSig, "signed because a secret was configured")
		assert.Contains(t, string(gotBody), "order.completed")
	})

	t.Run("transport error is surfaced", func(t *testing.T) {
		svc := newWebhookSubscriptionService(&fakeWebhookSubRepo{}, nil)

		err := svc.SendWebhook(context.Background(), port.OutgoingWebhookPayload{
			WebhookSubscription: domain.WebhookSubscription{URL: "http://127.0.0.1:0/unreachable"},
			Event:               port.PubSubPayload{Topic: "order.completed"},
		})

		require.Error(t, err)
	})
}
