//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"getpaidhq/internal/core/port"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"
)

func newCampaign(orgId, subId, custId string) domain.DunningCampaign {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return domain.DunningCampaign{
		OrgId:                orgId,
		Id:                   lib.GenerateId("dun"),
		SubscriptionId:       subId,
		CustomerId:           custId,
		WorkflowId:           lib.GenerateId("wf"),
		Status:               domain.DunningStatusActive,
		FailedAmount:         1999,
		Currency:             "USD",
		InitialFailureReason: "card_declined",
		StartedAt:            now,
		ConfigSnapshot:       map[string]any{"max_attempts": float64(5)},
		Metadata:             map[string]string{"k": "v"},
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func TestDunningRepo_Campaigns(t *testing.T) {
	db := testDB(t)
	repo := NewDunningRepo(db)
	ctx := context.Background()

	t.Run("Create then FindById round-trips", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		subId := lib.GenerateId("sub")
		c := newCampaign(orgId, subId, lib.GenerateId("cus"))

		created, err := repo.CreateCampaign(ctx, c)
		require.NoError(t, err)
		assert.Equal(t, c.Id, created.Id)
		assert.Equal(t, domain.DunningStatusActive, created.Status)
		assert.Equal(t, "card_declined", created.InitialFailureReason)
		assert.Equal(t, map[string]any{"max_attempts": float64(5)}, created.ConfigSnapshot)

		got, err := repo.FindCampaignById(ctx, orgId, c.Id)
		require.NoError(t, err)
		assert.Equal(t, c.Id, got.Id)
	})

	t.Run("Update mutates status", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		c, err := repo.CreateCampaign(ctx, newCampaign(orgId, lib.GenerateId("sub"), lib.GenerateId("cus")))
		require.NoError(t, err)

		c.Status = domain.DunningStatusRecovered
		c.RecoveredAmount = 1999
		updated, err := repo.UpdateCampaign(ctx, c)
		require.NoError(t, err)
		assert.Equal(t, domain.DunningStatusRecovered, updated.Status)
		assert.Equal(t, int64(1999), updated.RecoveredAmount)
	})

	t.Run("FindCampaignById not-found returns ErrRecordNotFound", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		_, err := repo.FindCampaignById(ctx, orgId, "missing")
		assert.True(t, errors.Is(err, port.ErrNotFound))
	})

	t.Run("FindCampaignsBySubscriptionId filters by subscription", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		subId := lib.GenerateId("sub")
		for range 2 {
			_, err := repo.CreateCampaign(ctx, newCampaign(orgId, subId, lib.GenerateId("cus")))
			require.NoError(t, err)
		}
		_, err := repo.CreateCampaign(ctx, newCampaign(orgId, lib.GenerateId("sub"), lib.GenerateId("cus")))
		require.NoError(t, err)

		p := domain.Pagination{Limit: 10, SortBy: "created_at", SortDirection: "asc"}
		cs, count, err := repo.FindCampaignsBySubscriptionId(ctx, orgId, subId, p)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Len(t, cs, 2)
	})

	t.Run("FindActiveCampaignForSubscription returns active/paused only", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		subId := lib.GenerateId("sub")

		// A closed campaign should be ignored.
		closed := newCampaign(orgId, subId, lib.GenerateId("cus"))
		closed.Status = domain.DunningStatusFailed
		_, err := repo.CreateCampaign(ctx, closed)
		require.NoError(t, err)

		active := newCampaign(orgId, subId, lib.GenerateId("cus"))
		active.Status = domain.DunningStatusActive
		createdActive, err := repo.CreateCampaign(ctx, active)
		require.NoError(t, err)

		got, err := repo.FindActiveCampaignForSubscription(ctx, orgId, subId)
		require.NoError(t, err)
		assert.Equal(t, createdActive.Id, got.Id)
	})

	t.Run("org-scoping isolates campaigns", func(t *testing.T) {
		orgA := uniqueOrg(t)
		orgB := uniqueOrg(t)
		cleanupOrg(t, db, orgA)
		cleanupOrg(t, db, orgB)
		created, err := repo.CreateCampaign(ctx, newCampaign(orgA, lib.GenerateId("sub"), lib.GenerateId("cus")))
		require.NoError(t, err)

		_, err = repo.FindCampaignById(ctx, orgB, created.Id)
		assert.True(t, errors.Is(err, port.ErrNotFound))
	})
}

func TestDunningRepo_Attempts(t *testing.T) {
	db := testDB(t)
	repo := NewDunningRepo(db)
	ctx := context.Background()

	t.Run("Create, FindById, FindByCampaignId", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		campaign, err := repo.CreateCampaign(ctx, newCampaign(orgId, lib.GenerateId("sub"), lib.GenerateId("cus")))
		require.NoError(t, err)

		a := domain.DunningAttempt{
			OrgId:             orgId,
			Id:                lib.GenerateId("att"),
			DunningCampaignId: campaign.Id,
			SubscriptionId:    campaign.SubscriptionId,
			AttemptNumber:     1,
			AttemptType:       domain.DunningAttemptTypeImmediate,
			Amount:            1999,
			Currency:          "USD",
			Status:            domain.PaymentStatusFailed,
			FailureReason:     "insufficient_funds",
			ProcessorResponse: map[string]any{"code": "51"},
			AttemptedAt:       time.Now().UTC().Truncate(time.Microsecond),
			CreatedAt:         time.Now().UTC().Truncate(time.Microsecond),
		}
		created, err := repo.CreateAttempt(ctx, a)
		require.NoError(t, err)
		assert.Equal(t, a.Id, created.Id)
		assert.Equal(t, domain.DunningAttemptTypeImmediate, created.AttemptType)
		assert.Equal(t, map[string]any{"code": "51"}, created.ProcessorResponse)

		got, err := repo.FindAttemptById(ctx, orgId, a.Id)
		require.NoError(t, err)
		assert.Equal(t, a.Id, got.Id)

		p := domain.Pagination{Limit: 10, SortBy: "attempt_number", SortDirection: "asc"}
		attempts, count, err := repo.FindAttemptsByCampaignId(ctx, orgId, campaign.Id, p)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		require.Len(t, attempts, 1)
		assert.Equal(t, a.Id, attempts[0].Id)
	})

	t.Run("FindAttemptById not-found returns ErrRecordNotFound", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		_, err := repo.FindAttemptById(ctx, orgId, "missing")
		assert.True(t, errors.Is(err, port.ErrNotFound))
	})
}

func TestDunningRepo_Tokens(t *testing.T) {
	db := testDB(t)
	repo := NewDunningRepo(db)
	ctx := context.Background()

	t.Run("Create, FindById, Update", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		subId := lib.GenerateId("sub")

		tok := domain.PaymentUpdateToken{
			OrgId:          orgId,
			TokenId:        lib.GenerateId("tok"),
			SubscriptionId: subId,
			CustomerId:     lib.GenerateId("cus"),
			Signature:      "sig123",
			ExpiresAt:      time.Now().UTC().Add(72 * time.Hour).Truncate(time.Microsecond),
			MaxUses:        5,
			UsedCount:      0,
			Status:         domain.TokenStatusActive,
			AllowedActions: map[string]bool{"update_payment_method": true},
			CreatedAt:      time.Now().UTC().Truncate(time.Microsecond),
		}
		created, err := repo.CreateToken(ctx, tok)
		require.NoError(t, err)
		assert.Equal(t, tok.TokenId, created.TokenId)
		assert.Equal(t, domain.TokenStatusActive, created.Status)
		assert.Equal(t, map[string]bool{"update_payment_method": true}, created.AllowedActions)

		got, err := repo.FindTokenById(ctx, orgId, tok.TokenId)
		require.NoError(t, err)
		assert.Equal(t, tok.TokenId, got.TokenId)

		// Update: consume a use and revoke.
		got.UsedCount = 1
		got.Status = domain.TokenStatusRevoked
		updated, err := repo.UpdateToken(ctx, got)
		require.NoError(t, err)
		assert.Equal(t, 1, updated.UsedCount)
		assert.Equal(t, domain.TokenStatusRevoked, updated.Status)
	})

	t.Run("FindTokensBySubscriptionId filters by subscription", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		subId := lib.GenerateId("sub")
		for range 2 {
			tok := domain.PaymentUpdateToken{
				OrgId:          orgId,
				TokenId:        lib.GenerateId("tok"),
				SubscriptionId: subId,
				CustomerId:     lib.GenerateId("cus"),
				Status:         domain.TokenStatusActive,
				ExpiresAt:      time.Now().UTC().Add(time.Hour),
				CreatedAt:      time.Now().UTC().Truncate(time.Microsecond),
			}
			_, err := repo.CreateToken(ctx, tok)
			require.NoError(t, err)
		}
		p := domain.Pagination{Limit: 10, SortBy: "created_at", SortDirection: "asc"}
		ts, count, err := repo.FindTokensBySubscriptionId(ctx, orgId, subId, p)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Len(t, ts, 2)
	})

	t.Run("FindTokenById not-found returns ErrRecordNotFound", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		_, err := repo.FindTokenById(ctx, orgId, "missing")
		assert.True(t, errors.Is(err, port.ErrNotFound))
	})
}

func TestDunningRepo_Configurations(t *testing.T) {
	db := testDB(t)
	repo := NewDunningRepo(db)
	ctx := context.Background()

	newConfig := func(orgId string, priority int, status domain.ConfigStatus) domain.DunningConfiguration {
		now := time.Now().UTC().Truncate(time.Microsecond)
		return domain.DunningConfiguration{
			OrgId:     orgId,
			Id:        lib.GenerateId("cfg"),
			Name:      "cfg-" + lib.GenerateId("n"),
			Priority:  priority,
			AppliesTo: domain.DunningConfigScopeOrganization,
			Config:    map[string]any{"max_attempts": float64(3)},
			Status:    status,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	t.Run("Create, FindById, Update", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		c, err := repo.CreateConfiguration(ctx, newConfig(orgId, 10, domain.ConfigStatusActive))
		require.NoError(t, err)
		assert.Equal(t, domain.ConfigStatusActive, c.Status)
		assert.Equal(t, map[string]any{"max_attempts": float64(3)}, c.Config)

		got, err := repo.FindConfigurationById(ctx, orgId, c.Id)
		require.NoError(t, err)
		assert.Equal(t, c.Id, got.Id)

		got.Status = domain.ConfigStatusArchived
		updated, err := repo.UpdateConfiguration(ctx, got)
		require.NoError(t, err)
		assert.Equal(t, domain.ConfigStatusArchived, updated.Status)
	})

	t.Run("FindConfigurationsByPriority returns active sorted desc", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		_, err := repo.CreateConfiguration(ctx, newConfig(orgId, 1, domain.ConfigStatusActive))
		require.NoError(t, err)
		_, err = repo.CreateConfiguration(ctx, newConfig(orgId, 100, domain.ConfigStatusActive))
		require.NoError(t, err)
		// Inactive must be excluded.
		_, err = repo.CreateConfiguration(ctx, newConfig(orgId, 50, domain.ConfigStatusInactive))
		require.NoError(t, err)

		cs, err := repo.FindConfigurationsByPriority(ctx, orgId)
		require.NoError(t, err)
		require.Len(t, cs, 2)
		assert.Equal(t, 100, cs[0].Priority, "highest priority first")
		assert.Equal(t, 1, cs[1].Priority)
	})

	t.Run("FindConfigurations paginates and counts", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		for range 3 {
			_, err := repo.CreateConfiguration(ctx, newConfig(orgId, 1, domain.ConfigStatusActive))
			require.NoError(t, err)
		}
		p := domain.Pagination{Limit: 2, SortBy: "priority", SortDirection: "desc"}
		cs, count, err := repo.FindConfigurations(ctx, orgId, p)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
		assert.Len(t, cs, 2)
	})
}

func TestDunningRepo_CustomerHistory(t *testing.T) {
	db := testDB(t)
	repo := NewDunningRepo(db)
	ctx := context.Background()

	t.Run("Get on missing returns empty (not error)", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		custId := lib.GenerateId("cus")
		h, err := repo.GetCustomerDunningHistory(ctx, orgId, custId)
		require.NoError(t, err)
		assert.Equal(t, orgId, h.OrgId)
		assert.Equal(t, custId, h.CustomerId)
		assert.Equal(t, 0, h.TotalDunningCampaigns)
	})

	t.Run("Upsert then Get round-trips", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		custId := lib.GenerateId("cus")
		h := domain.CustomerDunningHistory{
			OrgId:                 orgId,
			CustomerId:            custId,
			TotalDunningCampaigns: 2,
			SuccessfulRecoveries:  1,
			TotalAmountRecovered:  1999,
			UpdatedAt:             time.Now().UTC().Truncate(time.Microsecond),
		}
		saved, err := repo.UpsertCustomerDunningHistory(ctx, h)
		require.NoError(t, err)
		assert.Equal(t, 2, saved.TotalDunningCampaigns)

		got, err := repo.GetCustomerDunningHistory(ctx, orgId, custId)
		require.NoError(t, err)
		assert.Equal(t, 1, got.SuccessfulRecoveries)
		assert.Equal(t, int64(1999), got.TotalAmountRecovered)
	})
}
