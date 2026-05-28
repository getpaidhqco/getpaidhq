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

// fakePspRepo records PSP config creation and serves a configurable lookup.
type fakePspRepo struct {
	port.PspRepository
	byId      domain.PspConfig
	byIdErr   error
	createErr error
	created   []domain.PspConfig
}

func (r *fakePspRepo) Create(_ context.Context, c domain.PspConfig) (domain.PspConfig, error) {
	if r.createErr != nil {
		return domain.PspConfig{}, r.createErr
	}
	r.created = append(r.created, c)
	return c, nil
}

func (r *fakePspRepo) FindById(_ context.Context, _, _ string) (domain.PspConfig, error) {
	if r.byIdErr != nil {
		return domain.PspConfig{}, r.byIdErr
	}
	return r.byId, nil
}

// fakeSettingRepoRW records setting creation and serves a configurable lookup.
type fakeSettingRepoRW struct {
	port.SettingRepository
	byId      domain.Setting
	byIdErr   error
	createErr error
	created   []domain.Setting
}

func (r *fakeSettingRepoRW) Create(_ context.Context, s domain.Setting) (domain.Setting, error) {
	if r.createErr != nil {
		return domain.Setting{}, r.createErr
	}
	r.created = append(r.created, s)
	return s, nil
}

func (r *fakeSettingRepoRW) FindById(_ context.Context, _, _, _ string) (domain.Setting, error) {
	if r.byIdErr != nil {
		return domain.Setting{}, r.byIdErr
	}
	return r.byId, nil
}

func TestPspService_CreateGateway(t *testing.T) {
	t.Run("creates the psp config and a settings row", func(t *testing.T) {
		psp := &fakePspRepo{}
		settings := &fakeSettingRepoRW{}
		svc := NewPspService(psp, settings, silentLogger{}, &recordingPubSub{})

		got, err := svc.CreateGateway(context.Background(), port.CreateGatewayInput{
			OrgId: "org_1", PspId: domain.Paystack, Name: "Primary",
			Settings: map[string]string{"secret_key": "sk_test"},
		})

		require.NoError(t, err)
		assert.True(t, got.Active, "newly created gateway is active")
		assert.Equal(t, domain.Paystack, got.PspId)
		require.Len(t, psp.created, 1)
		require.Len(t, settings.created, 1)
		// The settings row is parented to the new psp id under the "settings" key.
		assert.Equal(t, got.Id, settings.created[0].ParentId)
		assert.Equal(t, "settings", settings.created[0].Id)
		assert.Equal(t, "psp", settings.created[0].Type)
		assert.Contains(t, settings.created[0].Value, "sk_test")
	})

	t.Run("psp create failure short-circuits before settings", func(t *testing.T) {
		psp := &fakePspRepo{createErr: errors.New("db down")}
		settings := &fakeSettingRepoRW{}
		svc := NewPspService(psp, settings, silentLogger{}, &recordingPubSub{})

		_, err := svc.CreateGateway(context.Background(), port.CreateGatewayInput{OrgId: "org_1", PspId: domain.Paystack})

		require.Error(t, err)
		assert.Empty(t, settings.created, "no settings written when psp create fails")
	})

	t.Run("settings create failure is surfaced", func(t *testing.T) {
		psp := &fakePspRepo{}
		settings := &fakeSettingRepoRW{createErr: errors.New("db down")}
		svc := NewPspService(psp, settings, silentLogger{}, &recordingPubSub{})

		_, err := svc.CreateGateway(context.Background(), port.CreateGatewayInput{OrgId: "org_1", PspId: domain.Paystack})

		require.Error(t, err)
	})
}
