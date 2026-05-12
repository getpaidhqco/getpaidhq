package domain

type EntityKey struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"`
}

type Pagination struct {
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
	SortDirection string `json:"sort_order"`
	SortBy        string `json:"sort_by"`
}

type MetadataUpdater interface {
	SetMetadata(meta map[string]string) MetadataUpdater
}

// Gateway represents a payment service provider identifier.
type Gateway string

const (
	CheckoutDotCom Gateway = "CheckoutDotCom"
	Paystack       Gateway = "Paystack"
)

// Entity and Operation types for data-change events.
type Entity string

const (
	CustomerEntity       Entity = "customers"
	SubscriptionEntity   Entity = "subscriptions"
	PaymentEntity        Entity = "payments"
	RefundEntity         Entity = "refunds"
	OrderEntity          Entity = "orders"
	ProductEntity        Entity = "products"
	CustomerCohortEntity Entity = "customer_cohorts"
)

type Operation string

const (
	InsertOperation   Operation = "INSERT"
	UpdateOperation   Operation = "UPDATE"
	DeleteOperation   Operation = "DELETE"
	TruncateOperation Operation = "TRUNCATE"
)
