package mocks

import (
	"context"
	"payloop/internal/domain/entities"
)

// MockDiscountService provides a reusable mock for discount service
type MockDiscountService struct{}

// NewMockDiscountService creates a new mock discount service
func NewMockDiscountService() *MockDiscountService {
	return &MockDiscountService{}
}

func (m *MockDiscountService) CalculateDiscount(ctx context.Context, orgId string, subscription entities.Subscription, amount int64) (int64, error) {
	return 0, nil
}