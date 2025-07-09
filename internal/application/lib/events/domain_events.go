package events

import (
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
	"time"
)

// BaseEvent is the foundation for all domain events
type BaseEvent struct {
	EventId          string            `json:"event_id"`
	EventType        string            `json:"event_type"`
	OrgId            string            `json:"org_id"`
	AggregateId      string            `json:"aggregate_id"`
	AggregateType    string            `json:"aggregate_type"`
	AggregateVersion int               `json:"aggregate_version"`
	Timestamp        time.Time         `json:"timestamp"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// NewBaseEvent creates a new base event with common fields populated
func NewBaseEvent(orgId, eventType, aggregateId, aggregateType string) BaseEvent {
	return BaseEvent{
		EventId:          GenerateEventId(),
		EventType:        eventType,
		OrgId:            orgId,
		AggregateId:      aggregateId,
		AggregateType:    aggregateType,
		AggregateVersion: 1,
		Timestamp:        time.Now().UTC(),
		Metadata:         make(map[string]string),
	}
}

// GenerateEventId generates a unique ID for events
func GenerateEventId() string {
	return lib.GenerateId("evt")
}

// BillingEvent represents a billing-related event
type BillingEvent struct {
	BaseEvent
	BillingEventType   string    `json:"billing_event_type"` // "invoice_created", "payment_charged", etc.
	SubscriptionId     string    `json:"subscription_id"`
	CustomerId         string    `json:"customer_id"`
	InvoiceId          string    `json:"invoice_id,omitempty"`
	Amount             int64     `json:"amount"`
	Currency           string    `json:"currency"`
	BillingPeriodStart time.Time `json:"billing_period_start"`
	BillingPeriodEnd   time.Time `json:"billing_period_end"`
	TaxAmount          int64     `json:"tax_amount,omitempty"`
	DiscountAmount     int64     `json:"discount_amount,omitempty"`
}

// SubscriptionEvent represents a subscription-related event
type SubscriptionEvent struct {
	BaseEvent
	SubscriptionEventType string                `json:"subscription_event_type"`
	SubscriptionId        string                `json:"subscription_id"`
	CustomerId            string                `json:"customer_id"`
	PreviousStatus        string                `json:"previous_status,omitempty"`
	NewStatus             string                `json:"new_status"`
	Subscription          entities.Subscription `json:"subscription"`
	ChangeReason          string                `json:"change_reason,omitempty"`
	EffectiveDate         time.Time             `json:"effective_date"`
}

// InvoiceEvent represents an invoice-related event
type InvoiceEvent struct {
	BaseEvent
	InvoiceEventType string           `json:"invoice_event_type"`
	InvoiceId        string           `json:"invoice_id"`
	SubscriptionId   string           `json:"subscription_id"`
	CustomerId       string           `json:"customer_id"`
	Invoice          entities.Invoice `json:"invoice"`
	Amount           int64            `json:"amount"`
	Currency         string           `json:"currency"`
	DueDate          time.Time        `json:"due_date"`
	PaidDate         *time.Time       `json:"paid_date,omitempty"`
}

// PaymentEvent represents a payment-related event
type PaymentEvent struct {
	BaseEvent
	PaymentEventType  string            `json:"payment_event_type"`
	PaymentId         string            `json:"payment_id"`
	SubscriptionId    string            `json:"subscription_id,omitempty"`
	CustomerId        string            `json:"customer_id"`
	InvoiceId         string            `json:"invoice_id,omitempty"`
	Payment           entities.Payment  `json:"payment"`
	Amount            int64             `json:"amount"`
	Currency          string            `json:"currency"`
	PaymentMethod     string            `json:"payment_method"`
	PaymentStatus     string            `json:"payment_status"`
	ProcessorResponse map[string]string `json:"processor_response,omitempty"`
	FailureReason     string            `json:"failure_reason,omitempty"`
}

// RefundEvent represents a refund-related event
type RefundEvent struct {
	BaseEvent
	RefundEventType   string            `json:"refund_event_type"`
	RefundId          string            `json:"refund_id"`
	PaymentId         string            `json:"payment_id"`
	SubscriptionId    string            `json:"subscription_id,omitempty"`
	CustomerId        string            `json:"customer_id"`
	Amount            int64             `json:"amount"`
	Currency          string            `json:"currency"`
	RefundReason      string            `json:"refund_reason"`
	RefundStatus      string            `json:"refund_status"`
	ProcessorResponse map[string]string `json:"processor_response,omitempty"`
}

// CustomerEvent represents a customer-related event
type CustomerEvent struct {
	BaseEvent
	CustomerEventType string            `json:"customer_event_type"`
	CustomerId        string            `json:"customer_id"`
	Customer          entities.Customer `json:"customer"`
	PreviousEmail     string            `json:"previous_email,omitempty"`
	NewEmail          string            `json:"new_email,omitempty"`
	ProfileChanges    map[string]string `json:"profile_changes,omitempty"`
}

// ProductEvent represents a product-related event
type ProductEvent struct {
	BaseEvent
	ProductEventType string            `json:"product_event_type"`
	ProductId        string            `json:"product_id"`
	Product          entities.Product  `json:"product"`
	PreviousState    *entities.Product `json:"previous_state,omitempty"`
}

// PriceEvent represents a price-related event
type PriceEvent struct {
	BaseEvent
	PriceEventType string          `json:"price_event_type"`
	PriceId        string          `json:"price_id"`
	ProductId      string          `json:"product_id"`
	Price          entities.Price  `json:"price"`
	PreviousState  *entities.Price `json:"previous_state,omitempty"`
}

// DunningEvent represents a dunning-related event
type DunningEvent struct {
	BaseEvent
	DunningEventType  string     `json:"dunning_event_type"`
	DunningCampaignId string     `json:"dunning_campaign_id"`
	SubscriptionId    string     `json:"subscription_id"`
	CustomerId        string     `json:"customer_id"`
	PaymentId         string     `json:"payment_id,omitempty"`
	AttemptNumber     int        `json:"attempt_number"`
	AttemptResult     string     `json:"attempt_result"`
	NextAttemptDate   *time.Time `json:"next_attempt_date,omitempty"`
	CampaignStatus    string     `json:"campaign_status"`
	CommunicationType string     `json:"communication_type,omitempty"`
	RecoveryAmount    int64      `json:"recovery_amount,omitempty"`
}

// NewPaymentEvent creates a new payment event
func NewPaymentEvent(orgId string, eventType string, payment entities.Payment) PaymentEvent {
	return PaymentEvent{
		BaseEvent:         NewBaseEvent(orgId, eventType, payment.Id, "payment"),
		PaymentEventType:  eventType,
		PaymentId:         payment.Id,
		SubscriptionId:    payment.SubscriptionId,
		CustomerId:        payment.OrderId, // Assuming customer is derived from order
		InvoiceId:         payment.InvoiceId,
		Payment:           payment,
		Amount:            payment.Amount,
		Currency:          payment.Currency,
		PaymentMethod:     string(payment.Psp),
		PaymentStatus:     string(payment.Status),
		ProcessorResponse: payment.Metadata,
	}
}

// NewSubscriptionEvent creates a new subscription event
func NewSubscriptionEvent(orgId string, eventType string, subscription entities.Subscription, previousStatus, newStatus, changeReason string) SubscriptionEvent {
	return SubscriptionEvent{
		BaseEvent:             NewBaseEvent(orgId, eventType, subscription.Id, "subscription"),
		SubscriptionEventType: eventType,
		SubscriptionId:        subscription.Id,
		CustomerId:            subscription.CustomerId,
		PreviousStatus:        previousStatus,
		NewStatus:             newStatus,
		Subscription:          subscription,
		ChangeReason:          changeReason,
		EffectiveDate:         time.Now().UTC(),
	}
}

// NewCustomerEvent creates a new customer event
func NewCustomerEvent(orgId string, eventType string, customer entities.Customer) CustomerEvent {
	return CustomerEvent{
		BaseEvent:         NewBaseEvent(orgId, eventType, customer.Id, "customer"),
		CustomerEventType: eventType,
		CustomerId:        customer.Id,
		Customer:          customer,
	}
}

// NewInvoiceEvent creates a new invoice event
func NewInvoiceEvent(orgId string, eventType string, invoice entities.Invoice) InvoiceEvent {
	return InvoiceEvent{
		BaseEvent:        NewBaseEvent(orgId, eventType, invoice.Id, "invoice"),
		InvoiceEventType: eventType,
		InvoiceId:        invoice.Id,
		SubscriptionId:   invoice.SubscriptionId,
		CustomerId:       invoice.CustomerId,
		Invoice:          invoice,
		Amount:           int64(invoice.Total),
		Currency:         invoice.Currency,
		DueDate:          invoice.DueAt,
	}
}

// NewBillingEvent creates a new billing event
func NewBillingEvent(orgId string, eventType string, subscriptionId, customerId, invoiceId string, amount int64, currency string) BillingEvent {
	return BillingEvent{
		BaseEvent:        NewBaseEvent(orgId, eventType, subscriptionId, "billing"),
		BillingEventType: eventType,
		SubscriptionId:   subscriptionId,
		CustomerId:       customerId,
		InvoiceId:        invoiceId,
		Amount:           amount,
		Currency:         currency,
	}
}
