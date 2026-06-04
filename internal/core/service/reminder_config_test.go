package service

import (
	"context"
	"testing"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"github.com/stretchr/testify/require"
)

// mapSettingRepo is a minimal in-memory port.SettingRepository backed by a map,
// keyed on (orgId, parentId, id). Missing keys return port.ErrNotFound — the
// same sentinel the postgres repo emits via translateErr — so the resolver's
// default-fallback branch is exercised faithfully.
type mapSettingRepo struct {
	items map[string]domain.Setting
}

func newMapSettingRepo() *mapSettingRepo {
	return &mapSettingRepo{items: map[string]domain.Setting{}}
}

func (r *mapSettingRepo) key(orgId, parentId, id string) string {
	return orgId + "|" + parentId + "|" + id
}

func (r *mapSettingRepo) FindById(_ context.Context, orgId, parentId, id string) (domain.Setting, error) {
	s, ok := r.items[r.key(orgId, parentId, id)]
	if !ok {
		return domain.Setting{}, port.ErrNotFound
	}
	return s, nil
}

func (r *mapSettingRepo) Create(_ context.Context, entity domain.Setting) (domain.Setting, error) {
	r.items[r.key(entity.OrgId, entity.ParentId, entity.Id)] = entity
	return entity, nil
}

func (r *mapSettingRepo) Upsert(_ context.Context, entity domain.Setting) (domain.Setting, error) {
	r.items[r.key(entity.OrgId, entity.ParentId, entity.Id)] = entity
	return entity, nil
}

func TestReminderConfigService_Resolve_DefaultWhenMissing(t *testing.T) {
	repo := newMapSettingRepo()
	svc := NewReminderConfigService(repo, silentLogger{})

	cfg, err := svc.ResolveReminderConfig(context.Background(), "org_x")
	require.NoError(t, err)
	require.Equal(t, domain.DefaultReminderConfig(), cfg)
}

func TestReminderConfigService_Resolve_DefaultWhenCorrupt(t *testing.T) {
	repo := newMapSettingRepo()
	// Seed a stored setting whose value is not valid JSON. Unlike the missing
	// case, FindById succeeds here, so this exercises the resolver's
	// parse-error→default branch (not the not-found branch).
	_, err := repo.Upsert(context.Background(), domain.Setting{
		OrgId:    "org_x",
		ParentId: domain.ReminderConfigSettingParent,
		Id:       domain.ReminderConfigSettingId,
		Value:    "{not valid json",
	})
	require.NoError(t, err)

	svc := NewReminderConfigService(repo, silentLogger{})

	cfg, err := svc.ResolveReminderConfig(context.Background(), "org_x")
	require.NoError(t, err)
	require.Equal(t, domain.DefaultReminderConfig(), cfg)
}

func TestReminderConfigService_SetThenResolve(t *testing.T) {
	repo := newMapSettingRepo()
	svc := NewReminderConfigService(repo, silentLogger{})

	want := domain.ReminderConfig{Enabled: true, Offsets: []time.Duration{24 * time.Hour}}
	require.NoError(t, svc.SetReminderConfig(context.Background(), "org_x", want))

	got, err := svc.ResolveReminderConfig(context.Background(), "org_x")
	require.NoError(t, err)
	require.Equal(t, want, got)
}
