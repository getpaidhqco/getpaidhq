package repositories

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/security"
	"time"
)

type PaymentMethodRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error)
	Create(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error)
	Update(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error)

	FindExpiringPaymentMethods(ctx context.Context, expiry time.Time) ([]entities.PaymentMethod, error)

	// Secure payment method operations
	FindSecureById(ctx context.Context, orgId string, id string, vault security.TokenVault) (entities.SecurePaymentMethod, error)
	CreateSecure(ctx context.Context, secureEntity entities.SecurePaymentMethod) (entities.SecurePaymentMethod, error)
	UpdateSecure(ctx context.Context, secureEntity entities.SecurePaymentMethod) (entities.SecurePaymentMethod, error)
}
