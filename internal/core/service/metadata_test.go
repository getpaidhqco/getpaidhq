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

type fakeMetadataRepo struct {
	port.MetadataStoreRepository
	byKey      domain.MetadataStore
	byKeyErr   error
	byParent   []domain.MetadataStore
	byType     []domain.MetadataStore
	byValue    []domain.MetadataStore
	noOrg      []domain.MetadataStore
	listErr    error
	createErr  error
	created    []domain.MetadataStore
	updated    []domain.MetadataStore
	deleted    []string
	deletedErr error
}

func (r *fakeMetadataRepo) Create(_ context.Context, m domain.MetadataStore) (domain.MetadataStore, error) {
	if r.createErr != nil {
		return domain.MetadataStore{}, r.createErr
	}
	r.created = append(r.created, m)
	return m, nil
}
func (r *fakeMetadataRepo) Update(_ context.Context, m domain.MetadataStore) (domain.MetadataStore, error) {
	r.updated = append(r.updated, m)
	return m, nil
}
func (r *fakeMetadataRepo) FindByKey(_ context.Context, _, _, _ string) (domain.MetadataStore, error) {
	if r.byKeyErr != nil {
		return domain.MetadataStore{}, r.byKeyErr
	}
	return r.byKey, nil
}
func (r *fakeMetadataRepo) FindByParent(_ context.Context, _, _ string) ([]domain.MetadataStore, error) {
	return r.byParent, r.listErr
}
func (r *fakeMetadataRepo) FindByParentType(_ context.Context, _, _, _ string) ([]domain.MetadataStore, error) {
	return r.byType, r.listErr
}
func (r *fakeMetadataRepo) FindByValue(_ context.Context, _, _, _ string) ([]domain.MetadataStore, error) {
	return r.byValue, r.listErr
}
func (r *fakeMetadataRepo) FindByValueWithoutOrg(_ context.Context, _, _, _ string) ([]domain.MetadataStore, error) {
	return r.noOrg, r.listErr
}
func (r *fakeMetadataRepo) Delete(_ context.Context, _, _, key string) error {
	if r.deletedErr != nil {
		return r.deletedErr
	}
	r.deleted = append(r.deleted, key)
	return nil
}

func TestMetadataService_Create(t *testing.T) {
	repo := &fakeMetadataRepo{}
	svc := NewMetadataService(repo, silentLogger{})

	got, err := svc.Create(context.Background(), port.CreateMetadataInput{
		OrgId: "org_1", ParentId: "cus_1", ParentType: "customer", Key: "k", Value: "v",
	})

	require.NoError(t, err)
	assert.Equal(t, "k", got.Key)
	require.Len(t, repo.created, 1)
	assert.Equal(t, "customer", repo.created[0].ParentType)
}

func TestMetadataService_Update(t *testing.T) {
	t.Run("preserves the existing parent type", func(t *testing.T) {
		repo := &fakeMetadataRepo{byKey: domain.MetadataStore{ParentType: "subscription"}}
		svc := NewMetadataService(repo, silentLogger{})

		got, err := svc.Update(context.Background(), port.UpdateMetadataInput{OrgId: "org_1", ParentId: "p", Key: "k", Value: "v2"})

		require.NoError(t, err)
		assert.Equal(t, "subscription", got.ParentType, "parent type carried over from existing record")
		require.Len(t, repo.updated, 1)
		assert.Equal(t, "v2", repo.updated[0].Value)
	})

	t.Run("missing record aborts the update", func(t *testing.T) {
		repo := &fakeMetadataRepo{byKeyErr: errors.New("missing")}
		svc := NewMetadataService(repo, silentLogger{})

		_, err := svc.Update(context.Background(), port.UpdateMetadataInput{OrgId: "org_1", ParentId: "p", Key: "k"})

		require.Error(t, err)
		assert.Empty(t, repo.updated)
	})
}

func TestMetadataService_Reads(t *testing.T) {
	repo := &fakeMetadataRepo{
		byKey:    domain.MetadataStore{Key: "k", Value: "v"},
		byParent: []domain.MetadataStore{{Key: "a"}, {Key: "b"}},
		byType:   []domain.MetadataStore{{Key: "c"}},
		byValue:  []domain.MetadataStore{{Key: "d"}},
		noOrg:    []domain.MetadataStore{{Key: "e"}, {Key: "f"}},
	}
	svc := NewMetadataService(repo, silentLogger{})
	ctx := context.Background()

	t.Run("GetByKey", func(t *testing.T) {
		got, err := svc.GetByKey(ctx, "org_1", "p", "k")
		require.NoError(t, err)
		assert.Equal(t, "v", got.Value)
	})
	t.Run("GetByParent", func(t *testing.T) {
		got, err := svc.GetByParent(ctx, "org_1", "p")
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
	t.Run("GetByParentType", func(t *testing.T) {
		got, err := svc.GetByParentType(ctx, "org_1", "customer", "k")
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})
	t.Run("GetByValue", func(t *testing.T) {
		got, err := svc.GetByValue(ctx, "org_1", "k", "v")
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})
	t.Run("GetByValueWithoutOrg", func(t *testing.T) {
		got, err := svc.GetByValueWithoutOrg(ctx, "k", "v", "customer")
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}

func TestMetadataService_Delete(t *testing.T) {
	t.Run("removes the entry", func(t *testing.T) {
		repo := &fakeMetadataRepo{}
		svc := NewMetadataService(repo, silentLogger{})

		err := svc.Delete(context.Background(), "org_1", "p", "k")
		require.NoError(t, err)
		assert.Equal(t, []string{"k"}, repo.deleted)
	})

	t.Run("surfaces repo error", func(t *testing.T) {
		repo := &fakeMetadataRepo{deletedErr: errors.New("db down")}
		svc := NewMetadataService(repo, silentLogger{})

		err := svc.Delete(context.Background(), "org_1", "p", "k")
		require.Error(t, err)
	})
}
