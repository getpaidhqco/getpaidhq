package models

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestPaymentLink_ToEntity_TokenHashHandling(t *testing.T) {
	// Test case 1: Non-empty TokenHash
	t.Run("Non-empty TokenHash", func(t *testing.T) {
		model := PaymentLink{
			OrgId:     "test_org",
			Id:        "test_id",
			Slug:      "test-slug",
			Data:      []byte(`{"test": "data"}`),
			Config:    []byte(`{"test": "config"}`),
			SingleUse: false,
			Status:    "active",
			TokenHash: pgtype.Text{String: "abc123hash", Valid: true},
		}

		entity := model.ToEntity()
		assert.Equal(t, "abc123hash", entity.TokenHash)
	})

	// Test case 2: Empty TokenHash (Valid = false)
	t.Run("Empty TokenHash (Valid = false)", func(t *testing.T) {
		model := PaymentLink{
			OrgId:     "test_org",
			Id:        "test_id",
			Slug:      "test-slug",
			Data:      []byte(`{"test": "data"}`),
			Config:    []byte(`{"test": "config"}`),
			SingleUse: false,
			Status:    "active",
			TokenHash: pgtype.Text{String: "", Valid: false},
		}

		entity := model.ToEntity()
		assert.Equal(t, "", entity.TokenHash)
	})

	// Test case 3: Empty TokenHash (Valid = true, but empty string)
	t.Run("Empty TokenHash (Valid = true, empty string)", func(t *testing.T) {
		model := PaymentLink{
			OrgId:     "test_org",
			Id:        "test_id",
			Slug:      "test-slug",
			Data:      []byte(`{"test": "data"}`),
			Config:    []byte(`{"test": "config"}`),
			SingleUse: false,
			Status:    "active",
			TokenHash: pgtype.Text{String: "", Valid: true},
		}

		entity := model.ToEntity()
		assert.Equal(t, "", entity.TokenHash)
	})
}