package common

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
