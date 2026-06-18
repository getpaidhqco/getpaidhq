package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// ---- fakes ----

// recordingPubSub captures every Publish so tests can assert which dunning
// topics fired and inspect the event payloads.
type recordingPubSub struct {
	mu        sync.Mutex
	published []publishedEvent
}

type publishedEvent struct {
	orgId   string
	topic   string
	message any
}

func (p *recordingPubSub) Publish(orgId, topic string, message any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.published = append(p.published, publishedEvent{orgId: orgId, topic: topic, message: message})
	return nil
}

func (p *recordingPubSub) Subscribe(topic string, _ func(string, []byte)) (port.PubSubSubscription, error) {
	return fakeSub{}, nil
}

func (p *recordingPubSub) Close() error { return nil }

func (p *recordingPubSub) byTopic(topic string) (publishedEvent, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.published {
		if e.topic == topic {
			return e, true
		}
	}
	return publishedEvent{}, false
}

func (p *recordingPubSub) hasTopic(topic string) bool {
	_, ok := p.byTopic(topic)
	return ok
}

// fakeDunningRepo embeds the port interface so it satisfies it with only the
// methods this test exercises overridden; any unexpected call nil-panics.
type fakeDunningRepo struct {
	port.DunningRepository
	campaign         domain.DunningCampaign
	findErr          error
	updated          []domain.DunningCampaign
	createdCampaigns []domain.DunningCampaign

	token        domain.PaymentUpdateToken
	tokenFindErr error
	tokenUpdates []domain.PaymentUpdateToken

	configs   []domain.DunningConfiguration
	configErr error
}

func (r *fakeDunningRepo) CreateCampaign(_ context.Context, c domain.DunningCampaign) (domain.DunningCampaign, error) {
	r.campaign = c
	r.createdCampaigns = append(r.createdCampaigns, c)
	return c, nil
}

func (r *fakeDunningRepo) FindCampaignById(_ context.Context, _, _ string) (domain.DunningCampaign, error) {
	if r.findErr != nil {
		return domain.DunningCampaign{}, r.findErr
	}
	return r.campaign, nil
}

func (r *fakeDunningRepo) UpdateCampaign(_ context.Context, c domain.DunningCampaign) (domain.DunningCampaign, error) {
	r.updated = append(r.updated, c)
	r.campaign = c // subsequent finds observe the mutation
	return c, nil
}

func (r *fakeDunningRepo) FindTokenById(_ context.Context, _, _ string) (domain.PaymentUpdateToken, error) {
	if r.tokenFindErr != nil {
		return domain.PaymentUpdateToken{}, r.tokenFindErr
	}
	return r.token, nil
}

func (r *fakeDunningRepo) UpdateToken(_ context.Context, t domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error) {
	r.tokenUpdates = append(r.tokenUpdates, t)
	r.token = t
	return t, nil
}

func (r *fakeDunningRepo) FindConfigurationsByPriority(_ context.Context, _ string) ([]domain.DunningConfiguration, error) {
	return r.configs, r.configErr
}

type fakeSubRepo struct {
	port.SubscriptionRepository
	sub       domain.Subscription
	byOrderId []domain.Subscription
	list      []domain.Subscription
	findErr   error
	createErr error
	updateErr error
	created   []domain.Subscription
	updated   []domain.Subscription
}

func (r *fakeSubRepo) FindByIdForUpdate(ctx context.Context, orgId, id string) (domain.Subscription, error) {
	return r.FindById(ctx, orgId, id)
}

func (r *fakeSubRepo) FindById(_ context.Context, _, _ string) (domain.Subscription, error) {
	if r.findErr != nil {
		return domain.Subscription{}, r.findErr
	}
	return r.sub, nil
}

func (r *fakeSubRepo) FindByOrderId(_ context.Context, _, _ string) ([]domain.Subscription, error) {
	return r.byOrderId, nil
}

func (r *fakeSubRepo) Create(_ context.Context, s domain.Subscription) (domain.Subscription, error) {
	if r.createErr != nil {
		return domain.Subscription{}, r.createErr
	}
	r.created = append(r.created, s)
	return s, nil
}

func (r *fakeSubRepo) Update(_ context.Context, s domain.Subscription) (domain.Subscription, error) {
	if r.updateErr != nil {
		return domain.Subscription{}, r.updateErr
	}
	r.updated = append(r.updated, s)
	r.sub = s
	return s, nil
}

func (r *fakeSubRepo) Find(_ context.Context, _ string, _ domain.Pagination) ([]domain.Subscription, int, error) {
	return r.list, len(r.list), nil
}

func newDunningServiceForTest(dr port.DunningRepository, sr port.SubscriptionRepository, ps port.PubSub) *DunningService {
	// customer/payment repos, subscription service and gateway factory are
	// unused by UpdateCampaignWithAttemptResult; ErrorReporter is a struct so a
	// zero value (also unused here) stands in.
	return NewDunningService(dr, sr, nil, nil, nil, nil, nil, ps, lib.ErrorReporter{}, silentLogger{})
}

// standardEscalation: suspend at attempt 3, final notice at 4, cancel at 5.
func standardEscalation() domain.DunningConfig {
	return domain.DunningConfig{
		EscalationRules: domain.EscalationRulesConfig{
			SuspendAfterAttempt: 3,
			FinalNoticeAttempt:  4,
			CancelAfterAttempt:  5,
		},
	}
}

// ---- escalation policy ----

func TestDunningService_UpdateCampaignWithAttemptResult(t *testing.T) {
	const (
		orgId      = "org_1"
		campaignId = "dc_1"
		subId      = "sub_1"
		customerId = "cust_1"
	)

	tests := []struct {
		name               string
		subStatus          domain.SubscriptionStatus
		attemptStatus      domain.PaymentStatus
		attemptCtx         domain.DunningAttemptContext
		wantCampaignStatus domain.DunningStatus
		wantSubUpdatedTo   domain.SubscriptionStatus // "" means: subscription must NOT be updated
		wantTopics         []string
		wantNoTopics       []string
	}{
		{
			name:               "success recovers campaign, no reactivation when not suspended",
			subStatus:          domain.SubscriptionStatusActive,
			attemptStatus:      domain.PaymentStatusSucceeded,
			attemptCtx:         domain.DunningAttemptContext{AttemptNumber: 1, WasSubscriptionSuspended: false},
			wantCampaignStatus: domain.DunningStatusRecovered,
			wantTopics:         []string{port.TopicDunningCampaignRecovered},
			wantNoTopics:       []string{port.TopicDunningSubscriptionReactivated},
		},
		{
			name:               "success reactivates a suspended subscription",
			subStatus:          domain.SubscriptionStatusUnpaid,
			attemptStatus:      domain.PaymentStatusSucceeded,
			attemptCtx:         domain.DunningAttemptContext{AttemptNumber: 4, WasSubscriptionSuspended: true},
			wantCampaignStatus: domain.DunningStatusRecovered,
			wantSubUpdatedTo:   domain.SubscriptionStatusActive,
			wantTopics:         []string{port.TopicDunningCampaignRecovered, port.TopicDunningSubscriptionReactivated},
		},
		{
			name:               "success does not reactivate an already-active subscription",
			subStatus:          domain.SubscriptionStatusActive,
			attemptStatus:      domain.PaymentStatusSucceeded,
			attemptCtx:         domain.DunningAttemptContext{AttemptNumber: 4, WasSubscriptionSuspended: true},
			wantCampaignStatus: domain.DunningStatusRecovered,
			wantTopics:         []string{port.TopicDunningCampaignRecovered},
			wantNoTopics:       []string{port.TopicDunningSubscriptionReactivated},
		},
		{
			name:               "failure below all thresholds just records the attempt",
			subStatus:          domain.SubscriptionStatusActive,
			attemptStatus:      domain.PaymentStatusFailed,
			attemptCtx:         domain.DunningAttemptContext{AttemptNumber: 1},
			wantCampaignStatus: domain.DunningStatusActive,
			wantTopics:         []string{port.TopicDunningAttemptFailed},
			wantNoTopics:       []string{port.TopicDunningSubscriptionSuspended, port.TopicDunningCampaignFailed},
		},
		{
			name:               "failure at suspend threshold suspends the subscription",
			subStatus:          domain.SubscriptionStatusActive,
			attemptStatus:      domain.PaymentStatusFailed,
			attemptCtx:         domain.DunningAttemptContext{AttemptNumber: 3},
			wantCampaignStatus: domain.DunningStatusActive,
			wantSubUpdatedTo:   domain.SubscriptionStatusUnpaid,
			wantTopics:         []string{port.TopicDunningSubscriptionSuspended, port.TopicDunningAttemptFailed},
		},
		{
			name:               "failure at suspend threshold is idempotent when already unpaid",
			subStatus:          domain.SubscriptionStatusUnpaid,
			attemptStatus:      domain.PaymentStatusFailed,
			attemptCtx:         domain.DunningAttemptContext{AttemptNumber: 3},
			wantCampaignStatus: domain.DunningStatusActive,
			wantTopics:         []string{port.TopicDunningAttemptFailed},
			wantNoTopics:       []string{port.TopicDunningSubscriptionSuspended},
		},
		{
			name:               "failure at cancel threshold cancels subscription and fails campaign",
			subStatus:          domain.SubscriptionStatusUnpaid,
			attemptStatus:      domain.PaymentStatusFailed,
			attemptCtx:         domain.DunningAttemptContext{AttemptNumber: 5},
			wantCampaignStatus: domain.DunningStatusFailed,
			wantSubUpdatedTo:   domain.SubscriptionStatusCancelled,
			wantTopics:         []string{port.TopicDunningCampaignFailed},
			wantNoTopics:       []string{port.TopicDunningAttemptFailed}, // cancel path returns before that publish
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := &fakeDunningRepo{campaign: domain.DunningCampaign{
				OrgId: orgId, Id: campaignId, SubscriptionId: subId, CustomerId: customerId,
				Status: domain.DunningStatusActive,
			}}
			sr := &fakeSubRepo{sub: domain.Subscription{OrgId: orgId, Id: subId, Status: tt.subStatus}}
			ps := &recordingPubSub{}
			svc := newDunningServiceForTest(dr, sr, ps)

			attempt := domain.DunningAttempt{
				OrgId: orgId, DunningCampaignId: campaignId, SubscriptionId: subId,
				Status: tt.attemptStatus, Amount: 5000,
			}

			got, err := svc.UpdateCampaignWithAttemptResult(context.Background(), attempt, standardEscalation(), tt.attemptCtx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCampaignStatus, got.Status, "returned campaign status")

			if tt.wantSubUpdatedTo == "" {
				assert.Empty(t, sr.updated, "subscription should not be updated")
			} else {
				require.NotEmpty(t, sr.updated, "subscription should be updated")
				assert.Equal(t, tt.wantSubUpdatedTo, sr.updated[len(sr.updated)-1].Status)
			}

			for _, topic := range tt.wantTopics {
				assert.Truef(t, ps.hasTopic(topic), "expected topic %q to be published", topic)
			}
			for _, topic := range tt.wantNoTopics {
				assert.Falsef(t, ps.hasTopic(topic), "did not expect topic %q to be published", topic)
			}
		})
	}
}

// The flag-bearing payloads deserve explicit assertions beyond "topic fired".
func TestDunningService_UpdateCampaignWithAttemptResult_EventPayloads(t *testing.T) {
	const (
		orgId      = "org_1"
		campaignId = "dc_1"
		subId      = "sub_1"
	)
	base := domain.DunningCampaign{OrgId: orgId, Id: campaignId, SubscriptionId: subId, Status: domain.DunningStatusActive}
	attempt := domain.DunningAttempt{OrgId: orgId, DunningCampaignId: campaignId, SubscriptionId: subId, Status: domain.PaymentStatusFailed}

	t.Run("suspension event carries the real old status", func(t *testing.T) {
		dr := &fakeDunningRepo{campaign: base}
		sr := &fakeSubRepo{sub: domain.Subscription{Id: subId, Status: domain.SubscriptionStatusActive}}
		ps := &recordingPubSub{}
		svc := newDunningServiceForTest(dr, sr, ps)

		_, err := svc.UpdateCampaignWithAttemptResult(context.Background(), attempt, standardEscalation(),
			domain.DunningAttemptContext{AttemptNumber: 3})
		require.NoError(t, err)

		ev, ok := ps.byTopic(port.TopicDunningSubscriptionSuspended)
		require.True(t, ok)
		sus := ev.message.(port.DunningSubscriptionEvent)
		assert.Equal(t, domain.SubscriptionStatusActive, sus.OldStatus)
		assert.Equal(t, domain.SubscriptionStatusUnpaid, sus.NewStatus)
	})

	t.Run("attempt_failed at final-notice carries shouldSuspend and isFinalNotice", func(t *testing.T) {
		// Already unpaid so no fresh suspension fires, but the flags on the
		// attempt_failed event must still reflect that we're past both thresholds.
		dr := &fakeDunningRepo{campaign: base}
		sr := &fakeSubRepo{sub: domain.Subscription{Id: subId, Status: domain.SubscriptionStatusUnpaid}}
		ps := &recordingPubSub{}
		svc := newDunningServiceForTest(dr, sr, ps)

		_, err := svc.UpdateCampaignWithAttemptResult(context.Background(), attempt, standardEscalation(),
			domain.DunningAttemptContext{AttemptNumber: 4})
		require.NoError(t, err)

		ev, ok := ps.byTopic(port.TopicDunningAttemptFailed)
		require.True(t, ok)
		failed := ev.message.(port.DunningAttemptEvent)
		assert.True(t, failed.ShouldSuspend, "past suspend threshold")
		assert.True(t, failed.IsFinalNotice, "at final-notice threshold")
		assert.False(t, ps.hasTopic(port.TopicDunningSubscriptionSuspended), "no fresh suspension when already unpaid")
	})
}

func TestDunningService_UpdateCampaignWithAttemptResult_CampaignNotFound(t *testing.T) {
	dr := &fakeDunningRepo{findErr: errors.New("nope")}
	sr := &fakeSubRepo{}
	ps := &recordingPubSub{}
	svc := newDunningServiceForTest(dr, sr, ps)

	_, err := svc.UpdateCampaignWithAttemptResult(context.Background(),
		domain.DunningAttempt{OrgId: "org_1", DunningCampaignId: "missing"}, standardEscalation(),
		domain.DunningAttemptContext{})

	require.Error(t, err)
	assert.Empty(t, ps.published, "no events on lookup failure")
}

// ---- token lifecycle ----

func TestDunningService_VerifyPaymentUpdateToken(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour)
	past := time.Now().UTC().Add(-time.Hour)

	tests := []struct {
		name           string
		token          domain.PaymentUpdateToken
		wantErr        bool
		wantStatusFlip domain.TokenStatus // "" means no UpdateToken persistence expected
	}{
		{
			name:  "active, unexpired, under max uses passes",
			token: domain.PaymentUpdateToken{Status: domain.TokenStatusActive, ExpiresAt: future, MaxUses: 5, UsedCount: 0},
		},
		{
			name:    "inactive token rejected without persistence",
			token:   domain.PaymentUpdateToken{Status: domain.TokenStatusRevoked, ExpiresAt: future, MaxUses: 5},
			wantErr: true,
		},
		{
			name:           "expired token flips to expired and persists",
			token:          domain.PaymentUpdateToken{Status: domain.TokenStatusActive, ExpiresAt: past, MaxUses: 5},
			wantErr:        true,
			wantStatusFlip: domain.TokenStatusExpired,
		},
		{
			name:           "exhausted token flips to max-uses and persists",
			token:          domain.PaymentUpdateToken{Status: domain.TokenStatusActive, ExpiresAt: future, MaxUses: 3, UsedCount: 3},
			wantErr:        true,
			wantStatusFlip: domain.TokenStatusMaxUsesReached,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := &fakeDunningRepo{token: tt.token}
			svc := newDunningServiceForTest(dr, &fakeSubRepo{}, &recordingPubSub{})

			_, err := svc.VerifyPaymentUpdateToken(context.Background(), "org_1", "tok_1")

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.wantStatusFlip == "" {
				assert.Empty(t, dr.tokenUpdates, "no token persistence expected")
			} else {
				require.NotEmpty(t, dr.tokenUpdates)
				assert.Equal(t, tt.wantStatusFlip, dr.tokenUpdates[len(dr.tokenUpdates)-1].Status)
			}
		})
	}
}

func TestDunningService_ActivatePaymentUpdateToken(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour)

	t.Run("increments use count and publishes activation", func(t *testing.T) {
		dr := &fakeDunningRepo{token: domain.PaymentUpdateToken{
			OrgId: "org_1", TokenId: "tok_1", Status: domain.TokenStatusActive, ExpiresAt: future, MaxUses: 5, UsedCount: 0,
		}}
		ps := &recordingPubSub{}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, ps)

		got, err := svc.ActivatePaymentUpdateToken(context.Background(), port.ActivatePaymentUpdateTokenInput{
			OrgId: "org_1", TokenId: "tok_1", UsedIp: "203.0.113.7",
		})

		require.NoError(t, err)
		assert.Equal(t, 1, got.UsedCount)
		assert.Equal(t, domain.TokenStatusActive, got.Status, "still active below max")
		assert.Equal(t, "203.0.113.7", got.LastUsedIp)
		assert.True(t, ps.hasTopic(port.TopicDunningTokenActivated))
	})

	t.Run("final use flips status to max-uses-reached", func(t *testing.T) {
		dr := &fakeDunningRepo{token: domain.PaymentUpdateToken{
			OrgId: "org_1", TokenId: "tok_1", Status: domain.TokenStatusActive, ExpiresAt: future, MaxUses: 5, UsedCount: 4,
		}}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, &recordingPubSub{})

		got, err := svc.ActivatePaymentUpdateToken(context.Background(), port.ActivatePaymentUpdateTokenInput{
			OrgId: "org_1", TokenId: "tok_1",
		})

		require.NoError(t, err)
		assert.Equal(t, 5, got.UsedCount)
		assert.Equal(t, domain.TokenStatusMaxUsesReached, got.Status)
	})

	t.Run("rejects an already-exhausted token without publishing", func(t *testing.T) {
		dr := &fakeDunningRepo{token: domain.PaymentUpdateToken{
			Status: domain.TokenStatusActive, ExpiresAt: future, MaxUses: 3, UsedCount: 3,
		}}
		ps := &recordingPubSub{}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, ps)

		_, err := svc.ActivatePaymentUpdateToken(context.Background(), port.ActivatePaymentUpdateTokenInput{OrgId: "org_1", TokenId: "tok_1"})

		require.Error(t, err)
		assert.False(t, ps.hasTopic(port.TopicDunningTokenActivated), "no activation event on rejected token")
	})
}

// ---- config resolution ----

func configWithCancelAfter(n int) domain.DunningConfig {
	c := domain.DunningConfig{}
	c.EscalationRules.CancelAfterAttempt = n
	return c
}

func mustConfigMap(t *testing.T, c domain.DunningConfig) map[string]any {
	t.Helper()
	m, err := configToMap(c)
	require.NoError(t, err)
	return m
}

func TestDunningService_ResolveConfig(t *testing.T) {
	t.Run("repo error falls back to default", func(t *testing.T) {
		dr := &fakeDunningRepo{configErr: errors.New("db down")}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, &recordingPubSub{})

		cfg, err := svc.ResolveConfig(context.Background(), "org_1")

		require.NoError(t, err)
		assert.Equal(t, domain.DefaultDunningConfig().EscalationRules.CancelAfterAttempt, cfg.EscalationRules.CancelAfterAttempt)
	})

	t.Run("empty list falls back to default", func(t *testing.T) {
		dr := &fakeDunningRepo{configs: nil}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, &recordingPubSub{})

		cfg, err := svc.ResolveConfig(context.Background(), "org_1")

		require.NoError(t, err)
		assert.Equal(t, domain.DefaultDunningConfig().EscalationRules.CancelAfterAttempt, cfg.EscalationRules.CancelAfterAttempt)
	})

	t.Run("returns the first decodable config", func(t *testing.T) {
		dr := &fakeDunningRepo{configs: []domain.DunningConfiguration{
			{Id: "cfg_1", Config: mustConfigMap(t, configWithCancelAfter(7))},
			{Id: "cfg_2", Config: mustConfigMap(t, configWithCancelAfter(9))},
		}}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, &recordingPubSub{})

		cfg, err := svc.ResolveConfig(context.Background(), "org_1")

		require.NoError(t, err)
		assert.Equal(t, 7, cfg.EscalationRules.CancelAfterAttempt)
	})

	t.Run("skips an undecodable config and returns the next", func(t *testing.T) {
		dr := &fakeDunningRepo{configs: []domain.DunningConfiguration{
			{Id: "cfg_bad", Config: map[string]any{"escalation_rules": "not-an-object"}},
			{Id: "cfg_good", Config: mustConfigMap(t, configWithCancelAfter(9))},
		}}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, &recordingPubSub{})

		cfg, err := svc.ResolveConfig(context.Background(), "org_1")

		require.NoError(t, err)
		assert.Equal(t, 9, cfg.EscalationRules.CancelAfterAttempt)
	})
}

func TestDunningService_LoadConfigForCampaign(t *testing.T) {
	t.Run("prefers the campaign snapshot over live config", func(t *testing.T) {
		dr := &fakeDunningRepo{
			campaign: domain.DunningCampaign{OrgId: "org_1", Id: "dc_1", ConfigSnapshot: mustConfigMap(t, configWithCancelAfter(2))},
			configs:  []domain.DunningConfiguration{{Id: "cfg_live", Config: mustConfigMap(t, configWithCancelAfter(8))}},
		}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, &recordingPubSub{})

		cfg, err := svc.LoadConfigForCampaign(context.Background(), "org_1", "dc_1")

		require.NoError(t, err)
		assert.Equal(t, 2, cfg.EscalationRules.CancelAfterAttempt, "snapshot wins")
	})

	t.Run("falls back to live config when no snapshot", func(t *testing.T) {
		dr := &fakeDunningRepo{
			campaign: domain.DunningCampaign{OrgId: "org_1", Id: "dc_1"}, // no snapshot
			configs:  []domain.DunningConfiguration{{Id: "cfg_live", Config: mustConfigMap(t, configWithCancelAfter(8))}},
		}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, &recordingPubSub{})

		cfg, err := svc.LoadConfigForCampaign(context.Background(), "org_1", "dc_1")

		require.NoError(t, err)
		assert.Equal(t, 8, cfg.EscalationRules.CancelAfterAttempt, "live config used")
	})

	t.Run("falls back to live config when snapshot is undecodable", func(t *testing.T) {
		dr := &fakeDunningRepo{
			campaign: domain.DunningCampaign{OrgId: "org_1", Id: "dc_1", ConfigSnapshot: map[string]any{"escalation_rules": "not-an-object"}},
			configs:  []domain.DunningConfiguration{{Id: "cfg_live", Config: mustConfigMap(t, configWithCancelAfter(8))}},
		}
		svc := newDunningServiceForTest(dr, &fakeSubRepo{}, &recordingPubSub{})

		cfg, err := svc.LoadConfigForCampaign(context.Background(), "org_1", "dc_1")

		require.NoError(t, err)
		assert.Equal(t, 8, cfg.EscalationRules.CancelAfterAttempt)
	})
}
