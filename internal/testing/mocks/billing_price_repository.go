package mocks

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/api/dto/request"
)

// MockBillingPriceRepository provides a reusable mock for price repository used in billing tests
type MockBillingPriceRepository struct{}

// NewMockBillingPriceRepository creates a new mock price repository for billing tests
func NewMockBillingPriceRepository() *MockBillingPriceRepository {
	return &MockBillingPriceRepository{}
}

func (m *MockBillingPriceRepository) FindById(ctx context.Context, orgId string, id string) (entities.Price, error) {
	return entities.Price{}, nil
}

func (m *MockBillingPriceRepository) Create(ctx context.Context, entity entities.Price) (entities.Price, error) {
	return entity, nil
}

func (m *MockBillingPriceRepository) Update(ctx context.Context, entity entities.Price) (entities.Price, error) {
	return entity, nil
}

func (m *MockBillingPriceRepository) FindByVariantId(ctx context.Context, orgId string, variantId string, p request.Pagination) ([]entities.Price, int, error) {
	return []entities.Price{}, 0, nil
}

func (m *MockBillingPriceRepository) GetPriceTiers(ctx context.Context, orgId string, priceId string) ([]entities.PriceTier, error) {
	return []entities.PriceTier{}, nil
}

func (m *MockBillingPriceRepository) Delete(ctx context.Context, orgId string, id string) error {
	return nil
}

func (m *MockBillingPriceRepository) CreatePriceTiers(ctx context.Context, tiers []entities.PriceTier) error {
	return nil
}

func (m *MockBillingPriceRepository) UpdatePriceTiers(ctx context.Context, orgId string, priceId string, tiers []entities.PriceTier) error {
	return nil
}

func (m *MockBillingPriceRepository) DeletePriceTiers(ctx context.Context, orgId string, priceId string) error {
	return nil
}