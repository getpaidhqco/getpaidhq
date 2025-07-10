package fixtures

import (
	"time"

	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/lib"
)

// SubscriptionBuilder provides a builder pattern for creating test subscriptions
type SubscriptionBuilder struct {
	subscription entities.Subscription
}

// NewSubscriptionBuilder creates a new subscription builder with sensible defaults
func NewSubscriptionBuilder() *SubscriptionBuilder {
	now := time.Now().UTC()

	subscription := entities.Subscription{
		OrgId:              "org_test123",
		Id:                 lib.GenerateId("sub"),
		Status:             entities.SubscriptionStatusActive,
		OrderId:            "order_test123",
		OrderItemId:        "item_test123",
		CustomerId:         "cust_test123",
		PaymentMethodId:    "pm_test123",
		StartDate:          now,
		BillingInterval:    prices.BillingIntervalMonth,
		BillingIntervalQty: 1,
		Cycles:             0, // 0 = unlimited
		BillingAnchor:      now.Day(),
		Currency:           "USD",
		Amount:             2500, // $25.00
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0), // 1 month from now
		RenewsAt:           now.AddDate(0, 1, 0),
		CyclesProcessed:    1,
		TotalRevenue:       2500,
		Metadata:           make(map[string]string),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Create a default subscription item
	subscriptionItem := entities.NewSubscriptionItem(
		subscription.OrgId,
		subscription.Id,
		"price_test123",
		"Test Subscription",
		"USD",
	)
	subscriptionItem.ProductId = "prod_test123"
	subscriptionItem.VariantId = "var_test123"
	subscriptionItem.Amount = 2500

	subscription.Items = []entities.SubscriptionItem{subscriptionItem}

	return &SubscriptionBuilder{
		subscription: subscription,
	}
}

// WithOrgId sets the organization ID
func (b *SubscriptionBuilder) WithOrgId(orgId string) *SubscriptionBuilder {
	b.subscription.OrgId = orgId
	return b
}

// WithId sets the subscription ID
func (b *SubscriptionBuilder) WithId(id string) *SubscriptionBuilder {
	b.subscription.Id = id
	return b
}

// WithStatus sets the subscription status
func (b *SubscriptionBuilder) WithStatus(status entities.SubscriptionStatus) *SubscriptionBuilder {
	b.subscription.Status = status
	return b
}

// WithCustomerId sets the customer ID
func (b *SubscriptionBuilder) WithCustomerId(customerId string) *SubscriptionBuilder {
	b.subscription.CustomerId = customerId
	return b
}

// WithProductVariantPrice sets the product, variant, and price IDs on the first subscription item
func (b *SubscriptionBuilder) WithProductVariantPrice(productId, variantId, priceId string) *SubscriptionBuilder {
	if len(b.subscription.Items) == 0 {
		// Create a subscription item if none exists
		subscriptionItem := entities.NewSubscriptionItem(
			b.subscription.OrgId,
			b.subscription.Id,
			priceId,
			"Test Subscription",
			b.subscription.Currency,
		)
		subscriptionItem.ProductId = productId
		subscriptionItem.VariantId = variantId
		subscriptionItem.Amount = b.subscription.Amount

		b.subscription.Items = []entities.SubscriptionItem{subscriptionItem}
	} else {
		// Update the first subscription item
		b.subscription.Items[0].ProductId = productId
		b.subscription.Items[0].VariantId = variantId
		b.subscription.Items[0].PriceId = priceId
	}
	return b
}

// WithAmount sets the subscription amount
func (b *SubscriptionBuilder) WithAmount(amount int64) *SubscriptionBuilder {
	b.subscription.Amount = amount
	return b
}

// WithBilling sets the billing interval and quantity
func (b *SubscriptionBuilder) WithBilling(interval prices.BillingInterval, qty int) *SubscriptionBuilder {
	b.subscription.BillingInterval = interval
	b.subscription.BillingIntervalQty = qty
	return b
}

// WithPeriod sets the current billing period
func (b *SubscriptionBuilder) WithPeriod(start, end time.Time) *SubscriptionBuilder {
	b.subscription.CurrentPeriodStart = start
	b.subscription.CurrentPeriodEnd = end
	b.subscription.RenewsAt = end
	return b
}

// WithMetadata adds metadata to the subscription
func (b *SubscriptionBuilder) WithMetadata(key, value string) *SubscriptionBuilder {
	if b.subscription.Metadata == nil {
		b.subscription.Metadata = make(map[string]string)
	}
	b.subscription.Metadata[key] = value
	return b
}

// WithTrial sets trial period
func (b *SubscriptionBuilder) WithTrial(endsAt time.Time) *SubscriptionBuilder {
	b.subscription.Status = entities.SubscriptionStatusTrial
	b.subscription.TrialEndsAt = endsAt
	return b
}

// Build returns the built subscription
func (b *SubscriptionBuilder) Build() entities.Subscription {
	return b.subscription
}

// CustomerBuilder provides a builder pattern for creating test customers
type CustomerBuilder struct {
	customer entities.Customer
}

// NewCustomerBuilder creates a new customer builder with sensible defaults
func NewCustomerBuilder() *CustomerBuilder {
	now := time.Now().UTC()

	return &CustomerBuilder{
		customer: entities.Customer{
			OrgId:     "org_test123",
			Id:        lib.GenerateId("cust"),
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Phone:     "+1234567890",
			Metadata:  make(map[string]string),
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// WithOrgId sets the organization ID
func (b *CustomerBuilder) WithOrgId(orgId string) *CustomerBuilder {
	b.customer.OrgId = orgId
	return b
}

// WithId sets the customer ID
func (b *CustomerBuilder) WithId(id string) *CustomerBuilder {
	b.customer.Id = id
	return b
}

// WithEmail sets the customer email
func (b *CustomerBuilder) WithEmail(email string) *CustomerBuilder {
	b.customer.Email = email
	return b
}

// WithName sets the customer name
func (b *CustomerBuilder) WithName(firstName, lastName string) *CustomerBuilder {
	b.customer.FirstName = firstName
	b.customer.LastName = lastName
	return b
}

// Build returns the built customer
func (b *CustomerBuilder) Build() entities.Customer {
	return b.customer
}

// VariantBuilder provides a builder pattern for creating test variants
type VariantBuilder struct {
	variant entities.Variant
}

// NewVariantBuilder creates a new variant builder with sensible defaults
func NewVariantBuilder() *VariantBuilder {
	now := time.Now().UTC()

	return &VariantBuilder{
		variant: entities.Variant{
			OrgId:       "org_test123",
			Id:          lib.GenerateId("var"),
			ProductId:   "prod_test123",
			Name:        "Test Variant",
			Description: "A test variant",
			Metadata:    make(map[string]string),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

// WithId sets the variant ID
func (b *VariantBuilder) WithId(id string) *VariantBuilder {
	b.variant.Id = id
	return b
}

// WithProductId sets the product ID
func (b *VariantBuilder) WithProductId(productId string) *VariantBuilder {
	b.variant.ProductId = productId
	return b
}

// WithName sets the variant name
func (b *VariantBuilder) WithName(name string) *VariantBuilder {
	b.variant.Name = name
	return b
}

// Build returns the built variant
func (b *VariantBuilder) Build() entities.Variant {
	return b.variant
}

// PriceBuilder provides a builder pattern for creating test prices
type PriceBuilder struct {
	price entities.Price
}

// NewPriceBuilder creates a new price builder with sensible defaults
func NewPriceBuilder() *PriceBuilder {
	now := time.Now().UTC()

	return &PriceBuilder{
		price: entities.Price{
			OrgId:              "org_test123",
			Id:                 lib.GenerateId("price"),
			VariantId:          "var_test123",
			Label:              "Monthly Plan",
			Category:           prices.PriceCategorySubscription,
			Scheme:             prices.Fixed,
			Currency:           common.USD,
			UnitPrice:          2500, // $25.00
			BillingInterval:    prices.BillingIntervalMonth,
			BillingIntervalQty: 1,
			Cycles:             0, // unlimited
			Metadata:           make(map[string]string),
			CreatedAt:          now,
			UpdatedAt:          now,
		},
	}
}

// WithId sets the price ID
func (b *PriceBuilder) WithId(id string) *PriceBuilder {
	b.price.Id = id
	return b
}

// WithVariantId sets the variant ID
func (b *PriceBuilder) WithVariantId(variantId string) *PriceBuilder {
	b.price.VariantId = variantId
	return b
}

// WithAmount sets the price amount
func (b *PriceBuilder) WithAmount(amount int64) *PriceBuilder {
	b.price.UnitPrice = amount
	return b
}

// WithBilling sets the billing interval and quantity
func (b *PriceBuilder) WithBilling(interval prices.BillingInterval, qty int) *PriceBuilder {
	b.price.BillingInterval = interval
	b.price.BillingIntervalQty = qty
	return b
}

// WithLabel sets the price label
func (b *PriceBuilder) WithLabel(label string) *PriceBuilder {
	b.price.Label = label
	return b
}

// Build returns the built price
func (b *PriceBuilder) Build() entities.Price {
	return b.price
}

// SubscriptionPlanChangeBuilder provides a builder for plan change records
type SubscriptionPlanChangeBuilder struct {
	planChange entities.SubscriptionPlanChange
}

// NewSubscriptionPlanChangeBuilder creates a new plan change builder
func NewSubscriptionPlanChangeBuilder() *SubscriptionPlanChangeBuilder {
	now := time.Now().UTC()

	return &SubscriptionPlanChangeBuilder{
		planChange: entities.SubscriptionPlanChange{
			Id:              lib.GenerateId("spc"),
			OrgId:           "org_test123",
			SubscriptionId:  "sub_test123",
			FromProductId:   "prod_test123",
			FromVariantId:   "var_old123",
			FromPriceId:     "price_old123",
			FromAmount:      2500,
			ToProductId:     "prod_test123",
			ToVariantId:     "var_new123",
			ToPriceId:       "price_new123",
			ToAmount:        4900,
			ChangeType:      "upgrade",
			EffectiveDate:   now,
			ProrationMode:   "immediate",
			ProrationAmount: 1200,
			Reason:          "Test plan change",
			InitiatedBy:     "customer",
			Metadata:        make(map[string]string),
			CreatedAt:       now,
		},
	}
}

// WithSubscriptionId sets the subscription ID
func (b *SubscriptionPlanChangeBuilder) WithSubscriptionId(subscriptionId string) *SubscriptionPlanChangeBuilder {
	b.planChange.SubscriptionId = subscriptionId
	return b
}

// WithFromPrice sets the from price details
func (b *SubscriptionPlanChangeBuilder) WithFromPrice(variantId, priceId string, amount int64) *SubscriptionPlanChangeBuilder {
	b.planChange.FromVariantId = variantId
	b.planChange.FromPriceId = priceId
	b.planChange.FromAmount = amount
	return b
}

// WithToPrice sets the to price details
func (b *SubscriptionPlanChangeBuilder) WithToPrice(variantId, priceId string, amount int64) *SubscriptionPlanChangeBuilder {
	b.planChange.ToVariantId = variantId
	b.planChange.ToPriceId = priceId
	b.planChange.ToAmount = amount
	return b
}

// WithChangeType sets the change type
func (b *SubscriptionPlanChangeBuilder) WithChangeType(changeType string) *SubscriptionPlanChangeBuilder {
	b.planChange.ChangeType = changeType
	return b
}

// Build returns the built plan change
func (b *SubscriptionPlanChangeBuilder) Build() entities.SubscriptionPlanChange {
	return b.planChange
}
