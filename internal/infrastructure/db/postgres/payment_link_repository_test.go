package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

func TestPaymentLinkRepository_TokenHashHandling(t *testing.T) {
	// Setup dependencies
	env := lib.NewEnv()
	logger := lib.GetLogger()
	db := NewDatabase(env.Get("DATABASE_URL"), logger)
	
	repo := NewPaymentLinkRepository(db, logger)
	
	ctx := context.Background()
	orgId := "test_org_id"
	
	// Test case 1: Create payment link with non-empty TokenHash
	t.Run("Create with non-empty TokenHash", func(t *testing.T) {
		paymentLink := entities.PaymentLink{
			OrgId:     orgId,
			Id:        "test_id_1",
			Slug:      "test-slug-1",
			Data:      map[string]interface{}{"test": "data"},
			Config:    map[string]interface{}{"test": "config"},
			SingleUse: false,
			Status:    "active",
			TokenHash: "abc123hash",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		created, err := repo.Create(ctx, paymentLink)
		assert.NoError(t, err)
		assert.Equal(t, "abc123hash", created.TokenHash)
		
		// Clean up
		_ = repo.Delete(ctx, orgId, "test_id_1")
	})
	
	// Test case 2: Create payment link with empty TokenHash
	t.Run("Create with empty TokenHash", func(t *testing.T) {
		paymentLink := entities.PaymentLink{
			OrgId:     orgId,
			Id:        "test_id_2",
			Slug:      "test-slug-2",
			Data:      map[string]interface{}{"test": "data"},
			Config:    map[string]interface{}{"test": "config"},
			SingleUse: false,
			Status:    "active",
			TokenHash: "", // Empty token hash
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		created, err := repo.Create(ctx, paymentLink)
		assert.NoError(t, err)
		assert.Equal(t, "", created.TokenHash)
		
		// Clean up
		_ = repo.Delete(ctx, orgId, "test_id_2")
	})
	
	// Test case 3: Update payment link TokenHash from empty to non-empty
	t.Run("Update TokenHash from empty to non-empty", func(t *testing.T) {
		// First create with empty TokenHash
		paymentLink := entities.PaymentLink{
			OrgId:     orgId,
			Id:        "test_id_3",
			Slug:      "test-slug-3",
			Data:      map[string]interface{}{"test": "data"},
			Config:    map[string]interface{}{"test": "config"},
			SingleUse: false,
			Status:    "active",
			TokenHash: "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		created, err := repo.Create(ctx, paymentLink)
		assert.NoError(t, err)
		assert.Equal(t, "", created.TokenHash)
		
		// Now update with non-empty TokenHash
		created.TokenHash = "updated_hash_123"
		updated, err := repo.Update(ctx, created)
		assert.NoError(t, err)
		assert.Equal(t, "updated_hash_123", updated.TokenHash)
		
		// Clean up
		_ = repo.Delete(ctx, orgId, "test_id_3")
	})
	
	// Test case 4: Update payment link TokenHash from non-empty to empty
	t.Run("Update TokenHash from non-empty to empty", func(t *testing.T) {
		// First create with non-empty TokenHash
		paymentLink := entities.PaymentLink{
			OrgId:     orgId,
			Id:        "test_id_4",
			Slug:      "test-slug-4",
			Data:      map[string]interface{}{"test": "data"},
			Config:    map[string]interface{}{"test": "config"},
			SingleUse: false,
			Status:    "active",
			TokenHash: "initial_hash_456",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		created, err := repo.Create(ctx, paymentLink)
		assert.NoError(t, err)
		assert.Equal(t, "initial_hash_456", created.TokenHash)
		
		// Now update with empty TokenHash
		created.TokenHash = ""
		updated, err := repo.Update(ctx, created)
		assert.NoError(t, err)
		assert.Equal(t, "", updated.TokenHash)
		
		// Clean up
		_ = repo.Delete(ctx, orgId, "test_id_4")
	})
}