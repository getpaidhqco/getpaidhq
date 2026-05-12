package port

import (
	"context"
	"payloop/internal/core/domain"
	"time"
)

// SubscriptionRepository manages subscription persistence.
type SubscriptionRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.Subscription, error)
	Create(ctx context.Context, entity domain.Subscription) (domain.Subscription, error)
	Update(ctx context.Context, entity domain.Subscription) (domain.Subscription, error)
	FindByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error)
	Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Subscription, int, error)
}

// OrderRepository manages order persistence.
// Also handles order items (merged from OrderItemRepository).
type OrderRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.Order, error)
	Create(ctx context.Context, entity domain.Order) (domain.Order, error)
	Update(ctx context.Context, entity domain.Order) (domain.Order, error)
	Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Order, int, error)

	// Order item operations (merged from OrderItemRepository)
	FindOrderItemById(ctx context.Context, orgId string, id string) (domain.OrderItem, error)
	CreateOrderItem(ctx context.Context, entity domain.OrderItem) (domain.OrderItem, error)
	UpdateOrderItem(ctx context.Context, orderItem domain.OrderItem) (domain.OrderItem, error)
	FindOrderItemsByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.OrderItem, error)
}

// CustomerRepository manages customer persistence.
// Also handles cohort operations (merged from CohortRepository).
type CustomerRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.Customer, error)
	FindByEmail(ctx context.Context, orgId string, email string) (domain.Customer, error)
	Create(ctx context.Context, entity domain.Customer) (domain.Customer, error)
	Update(ctx context.Context, entity domain.Customer) (domain.Customer, error)
	List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.Customer, int, error)
	FindPaymentMethodById(ctx context.Context, orgId string, id string) (domain.PaymentMethod, error)
	AddToCohort(ctx context.Context, orgId string, customerId string, cohortId string, cohortValue string) (domain.Customer, error)

	// Cohort operations (merged from CohortRepository)
	FindCohortById(ctx context.Context, orgId string, id string) (domain.Cohort, error)
	CreateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error)
	UpdateCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error)
	DeleteCohort(ctx context.Context, input domain.Cohort) (domain.Cohort, error)
}

// ProductRepository manages product persistence.
type ProductRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.Product, error)
	Create(ctx context.Context, product domain.Product) (domain.Product, error)
	Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Product, int, error)
	Update(ctx context.Context, product domain.Product) (domain.Product, error)
	Delete(ctx context.Context, orgId string, id string) error
}

// VariantRepository manages product variant persistence.
type VariantRepository interface {
	Create(ctx context.Context, entity domain.Variant) (domain.Variant, error)
	FindById(ctx context.Context, orgId string, id string) (domain.Variant, error)
	FindByProductId(ctx context.Context, orgId string, productId string, p domain.Pagination) ([]domain.Variant, int, error)
	Update(ctx context.Context, entity domain.Variant) (domain.Variant, error)
	Delete(ctx context.Context, orgId string, id string) error
}

// PriceRepository manages price persistence.
type PriceRepository interface {
	Create(ctx context.Context, entity domain.Price) (domain.Price, error)
	FindById(ctx context.Context, orgId string, id string) (domain.Price, error)
	FindByVariantId(ctx context.Context, orgId string, variantId string, p domain.Pagination) ([]domain.Price, int, error)
	Update(ctx context.Context, entity domain.Price) (domain.Price, error)
	Delete(ctx context.Context, orgId string, id string) error
}

// PaymentRepository manages payment persistence.
type PaymentRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.Payment, error)
	FindByPspId(ctx context.Context, orgId string, id string) (domain.Payment, error)
	ListByPspId(ctx context.Context, psp domain.Gateway, pspId string) ([]domain.Payment, error)
	FindBySubscriptionId(ctx context.Context, orgId string, id string, p domain.Pagination) ([]domain.Payment, int, error)
	Create(ctx context.Context, entity domain.Payment) (domain.Payment, error)
	Update(ctx context.Context, entity domain.Payment) (domain.Payment, error)
	CreateRefund(ctx context.Context, refund domain.Refund) (domain.Refund, error)
}

// PaymentMethodRepository manages payment method persistence.
type PaymentMethodRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.PaymentMethod, error)
	Create(ctx context.Context, entity domain.PaymentMethod) (domain.PaymentMethod, error)
	Update(ctx context.Context, entity domain.PaymentMethod) (domain.PaymentMethod, error)
	FindExpiringPaymentMethods(ctx context.Context, expiry time.Time) ([]domain.PaymentMethod, error)
}

// SessionRepository manages session persistence.
type SessionRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.Session, error)
	Create(ctx context.Context, input domain.Session) (domain.Session, error)
}

// CartRepository manages cart persistence.
type CartRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.Cart, error)
	Create(ctx context.Context, input domain.Cart) (domain.Cart, error)
	Update(ctx context.Context, input domain.Cart) (domain.Cart, error)
}

// OrgRepository manages organization persistence.
type OrgRepository interface {
	Create(ctx context.Context, entity domain.Org) (domain.Org, error)
}

// PspRepository manages payment service provider configuration persistence.
type PspRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.PspConfig, error)
	Create(ctx context.Context, input domain.PspConfig) (domain.PspConfig, error)
}

// SettingRepository manages settings persistence.
type SettingRepository interface {
	FindById(ctx context.Context, orgId string, parentId string, id string) (domain.Setting, error)
	Create(ctx context.Context, entity domain.Setting) (domain.Setting, error)
}

// ApiKeyRepository manages API key persistence.
type ApiKeyRepository interface {
	FindById(ctx context.Context, orgId string, id string) (domain.ApiKey, error)
	FindByKey(ctx context.Context, key string) (domain.ApiKey, error)
	Create(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error)
	Update(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error)
	Delete(ctx context.Context, orgId string, id string) error
}

// IdempotencyKeyRepository manages idempotency keys for preventing duplicate operations.
type IdempotencyKeyRepository interface {
	Exists(ctx context.Context, key string) (bool, error)
	Create(ctx context.Context, key string, expiresAt time.Time) error
}

// UserRepository manages user persistence.
type UserRepository any

// WebhookSubscriptionRepository manages webhook subscription persistence.
type WebhookSubscriptionRepository interface {
	Create(ctx context.Context, subscription domain.WebhookSubscription) (domain.WebhookSubscription, error)
	GetByID(ctx context.Context, orgId string, id string) (domain.WebhookSubscription, error)
	FindByEvent(ctx context.Context, orgId string, event string) ([]domain.WebhookSubscription, error)
	Update(ctx context.Context, subscription domain.WebhookSubscription) (domain.WebhookSubscription, error)
	Delete(ctx context.Context, id string) error
}

// MetadataStoreRepository manages key-value metadata persistence.
type MetadataStoreRepository interface {
	FindByKey(ctx context.Context, orgId string, parentId string, key string) (domain.MetadataStore, error)
	FindByParent(ctx context.Context, orgId string, parentId string) ([]domain.MetadataStore, error)
	FindByParentType(ctx context.Context, orgId string, parentType string, key string) ([]domain.MetadataStore, error)
	FindByValue(ctx context.Context, orgId string, key string, value string) ([]domain.MetadataStore, error)
	FindByValueWithoutOrg(ctx context.Context, key string, value string, parentType string) ([]domain.MetadataStore, error)
	Create(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error)
	Update(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error)
	Delete(ctx context.Context, orgId string, parentId string, key string) error
}

// ReportRepository manages reporting data and analytics queries.
type ReportRepository interface {
	GetMRR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error)
	GetARR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error)
	GetActiveSubscribers(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error)
	GetRefundTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error)
	GetCustomerChurnTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error)
	GetCustomerChurnRates(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error)
	UpsertSubscription(ctx context.Context, entity domain.Subscription) error
	UpsertPayment(ctx context.Context, entity domain.Payment) error
	UpsertCustomer(ctx context.Context, entity domain.Customer) error
	UpsertRefund(ctx context.Context, entity domain.Refund) error
	UpsertCustomerCohort(ctx context.Context, entity domain.CustomerCohort) error
	StoreDailyMetrics(ctx context.Context, input ProcessDailyMetricsInput) error
	ProcessDailyMetrics(ctx context.Context, d time.Time) error
}

// DataChangeEvent represents a data-change event for the reporting pipeline.
type DataChangeEvent struct {
	Operation domain.Operation
	Entity    domain.Entity
	NewObject any
	OldObject any
}

// ProcessDailyMetricsInput is the input for processing daily reporting metrics.
type ProcessDailyMetricsInput struct {
	OrgId    string    `json:"org_id"`
	Date     time.Time `json:"date"`
	Timezone string    `json:"timezone"`
}

// CreateGatewayInput is the input for creating a PSP gateway configuration.
type CreateGatewayInput struct {
	OrgId    string            `json:"org_id" validate:"required"`
	PspId    domain.Gateway    `json:"psp" validate:"required"`
	Name     string            `json:"name"`
	Settings map[string]string `json:"settings" validate:"required"`
}

// CreateOrgInput is the input for creating an organization.
type CreateOrgInput struct {
	Owner    AuthUser          `json:"owner"`
	Name     string            `json:"name"`
	Country  string            `json:"country"`
	Timezone string            `json:"timezone"`
	Metadata map[string]string `json:"metadata"`
}

// CreateMetadataInput represents the input for creating a metadata entry.
type CreateMetadataInput struct {
	OrgId      string `json:"org_id" binding:"required"`
	ParentId   string `json:"parent_id" binding:"required"`
	ParentType string `json:"parent_type" binding:"required"`
	Key        string `json:"key" binding:"required"`
	Value      string `json:"value" binding:"required"`
	Namespace  string `json:"namespace"`
}

// UpdateMetadataInput represents the input for updating a metadata entry.
type UpdateMetadataInput struct {
	OrgId     string `json:"org_id" binding:"required"`
	ParentId  string `json:"parent_id" binding:"required"`
	Key       string `json:"key" binding:"required"`
	Value     string `json:"value" binding:"required"`
	Namespace string `json:"namespace"`
}
