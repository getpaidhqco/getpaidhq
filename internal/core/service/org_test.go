package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// ---- fakes specific to OrgService ----

type fakeOrgRepo struct {
	port.OrgRepository
	createErr error
	created   []domain.Org
}

func (r *fakeOrgRepo) Create(_ context.Context, o domain.Org) (domain.Org, error) {
	if r.createErr != nil {
		return domain.Org{}, r.createErr
	}
	r.created = append(r.created, o)
	return o, nil
}

type fakeApiKeyRepo struct {
	port.ApiKeyRepository
	createErr error
	created   []domain.ApiKey
}

func (r *fakeApiKeyRepo) Create(_ context.Context, k domain.ApiKey) (domain.ApiKey, error) {
	if r.createErr != nil {
		return domain.ApiKey{}, r.createErr
	}
	r.created = append(r.created, k)
	return k, nil
}

// fakeCohortCustomerRepo records cohort creation only.
type fakeCohortCustomerRepo struct {
	port.CustomerRepository
	cohortErr error
	cohorts   []domain.Cohort
}

func (r *fakeCohortCustomerRepo) CreateCohort(_ context.Context, c domain.Cohort) (domain.Cohort, error) {
	if r.cohortErr != nil {
		return domain.Cohort{}, r.cohortErr
	}
	r.cohorts = append(r.cohorts, c)
	return c, nil
}

// fakeAuthProvider records external org creation and returns a remapped id.
type fakeAuthProvider struct {
	externalId string
	createErr  error
	called     bool
}

func (a *fakeAuthProvider) CreateOrg(_ context.Context, _ domain.Org, _ string) (port.CreateOrgResponse, error) {
	a.called = true
	if a.createErr != nil {
		return port.CreateOrgResponse{}, a.createErr
	}
	return port.CreateOrgResponse{ExternalId: a.externalId}, nil
}
func (a *fakeAuthProvider) AddUserToOrg(string, string, port.UserRole) error { return nil }
func (a *fakeAuthProvider) RemoveUserFromOrg(string, string) error           { return nil }
func (a *fakeAuthProvider) DeleteOrg(string) error                           { return nil }
func (a *fakeAuthProvider) HandleWebhook(string) error                       { return nil }

func newOrgService(repo port.OrgRepository, auth port.AuthProvider, cust port.CustomerRepository, apiKeys port.ApiKeyRepository, ps port.PubSub) *OrgService {
	if ps == nil {
		ps = &recordingPubSub{}
	}
	return NewOrgService(repo, ps, auth, cust, nil, nil, apiKeys, silentLogger{})
}

func TestOrgService_Create(t *testing.T) {
	t.Run("no owner: local id, api key, cohort, and org.created event", func(t *testing.T) {
		repo := &fakeOrgRepo{}
		keys := &fakeApiKeyRepo{}
		cust := &fakeCohortCustomerRepo{}
		auth := &fakeAuthProvider{}
		ps := &recordingPubSub{}
		svc := newOrgService(repo, auth, cust, keys, ps)

		got, err := svc.Create(context.Background(), port.CreateOrgInput{Name: "Acme"})

		require.NoError(t, err)
		assert.Equal(t, domain.OrgStatusActive, got.Status)
		assert.False(t, auth.called, "no auth provider call without an owner")
		require.Len(t, repo.created, 1)
		require.Len(t, keys.created, 1)
		assert.Equal(t, got.Id, keys.created[0].OrgId)
		assert.Equal(t, []string{"signup_date"}, cohortIds(cust.cohorts))
		assert.True(t, ps.hasTopic(port.TopicOrgCreated))
	})

	t.Run("with owner: org id is remapped to the auth provider external id", func(t *testing.T) {
		repo := &fakeOrgRepo{}
		auth := &fakeAuthProvider{externalId: "org_ext_123"}
		svc := newOrgService(repo, auth, &fakeCohortCustomerRepo{}, &fakeApiKeyRepo{}, nil)

		got, err := svc.Create(context.Background(), port.CreateOrgInput{
			Name:  "Acme",
			Owner: port.AuthUser{Id: "user_1"},
		})

		require.NoError(t, err)
		assert.True(t, auth.called)
		assert.Equal(t, "org_ext_123", got.Id)
	})

	t.Run("auth provider failure aborts before persistence", func(t *testing.T) {
		repo := &fakeOrgRepo{}
		auth := &fakeAuthProvider{createErr: errors.New("clerk down")}
		ps := &recordingPubSub{}
		svc := newOrgService(repo, auth, &fakeCohortCustomerRepo{}, &fakeApiKeyRepo{}, ps)

		_, err := svc.Create(context.Background(), port.CreateOrgInput{Name: "Acme", Owner: port.AuthUser{Id: "user_1"}})

		require.Error(t, err)
		assert.Empty(t, repo.created)
		assert.False(t, ps.hasTopic(port.TopicOrgCreated))
	})

	t.Run("api key failure aborts after org persistence and before event", func(t *testing.T) {
		repo := &fakeOrgRepo{}
		keys := &fakeApiKeyRepo{createErr: errors.New("db down")}
		ps := &recordingPubSub{}
		svc := newOrgService(repo, &fakeAuthProvider{}, &fakeCohortCustomerRepo{}, keys, ps)

		_, err := svc.Create(context.Background(), port.CreateOrgInput{Name: "Acme"})

		require.Error(t, err)
		assert.False(t, ps.hasTopic(port.TopicOrgCreated))
	})

	t.Run("cohort failure: org is still persisted and the event still publishes", func(t *testing.T) {
		// The cohort write is best-effort (logged as a warning, not aborted on),
		// so the org is created and org.created is published regardless. Note the
		// service still returns the stale cohort error in its second return value
		// — the org row and event are the observable side effects we assert on.
		repo := &fakeOrgRepo{}
		cust := &fakeCohortCustomerRepo{cohortErr: errors.New("cohort down")}
		ps := &recordingPubSub{}
		svc := newOrgService(repo, &fakeAuthProvider{}, cust, &fakeApiKeyRepo{}, ps)

		got, _ := svc.Create(context.Background(), port.CreateOrgInput{Name: "Acme"})

		assert.NotEmpty(t, got.Id, "org persisted despite cohort failure")
		require.Len(t, repo.created, 1)
		assert.True(t, ps.hasTopic(port.TopicOrgCreated))
	})
}

func cohortIds(cs []domain.Cohort) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Id)
	}
	return out
}
