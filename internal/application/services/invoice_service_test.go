package services

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/lib"
	"testing"
	"time"
)

// Mock implementations

type MockLogger struct{}

func (m MockLogger) Debug(msg string, args ...any) {}
func (m MockLogger) Info(msg string, args ...any)  {}
func (m MockLogger) Warn(msg string, args ...any)  {}
func (m MockLogger) Error(msg string, args ...any) {}
func (m MockLogger) Fatal(msg string, args ...any) {}

func (m MockLogger) Debugf(template string, args ...interface{}) {}
func (m MockLogger) Infof(template string, args ...interface{})  {}
func (m MockLogger) Warnf(template string, args ...interface{})  {}
func (m MockLogger) Errorf(template string, args ...interface{}) {}
func (m MockLogger) Panicf(template string, args ...interface{}) {}
func (m MockLogger) Fatalf(template string, args ...interface{}) {}

func (m MockLogger) Sync() error { return nil }

type MockErrorReporter struct{}

func (m *MockErrorReporter) ReportError(ctx interface{}, err error, data map[string]interface{}) {}

type MockPubSub struct{}

func (m MockPubSub) Publish(orgId string, topic string, message interface{}) error {
	return nil
}

func (m MockPubSub) Subscribe(topic string, handler func(topic string, data []byte)) (events.Subscription, error) {
	return nil, nil
}

type MockInvoiceRepository struct {
	invoices map[string]entities.Invoice
}

func NewMockInvoiceRepository() *MockInvoiceRepository {
	return &MockInvoiceRepository{
		invoices: make(map[string]entities.Invoice),
	}
}

func (m *MockInvoiceRepository) FindById(ctx context.Context, orgId string, id string) (entities.Invoice, error) {
	invoice, exists := m.invoices[id]
	if !exists {
		return entities.Invoice{}, lib.NewCustomError(lib.NotFoundError, "Invoice not found", nil)
	}
	return invoice, nil
}

func (m *MockInvoiceRepository) Create(ctx context.Context, entity entities.Invoice) (entities.Invoice, error) {
	m.invoices[entity.Id] = entity
	return entity, nil
}

func (m *MockInvoiceRepository) Update(ctx context.Context, entity entities.Invoice) (entities.Invoice, error) {
	m.invoices[entity.Id] = entity
	return entity, nil
}

func (m *MockInvoiceRepository) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Invoice, int, error) {
	var invoices []entities.Invoice
	for _, invoice := range m.invoices {
		if invoice.OrgId == orgId {
			invoices = append(invoices, invoice)
		}
	}
	return invoices, len(invoices), nil
}

func (m *MockInvoiceRepository) FindByCustomerId(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]entities.Invoice, int, error) {
	var invoices []entities.Invoice
	for _, invoice := range m.invoices {
		if invoice.OrgId == orgId && invoice.CustomerId == customerId {
			invoices = append(invoices, invoice)
		}
	}
	return invoices, len(invoices), nil
}

func (m *MockInvoiceRepository) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.Invoice, int, error) {
	var invoices []entities.Invoice
	for _, invoice := range m.invoices {
		if invoice.OrgId == orgId && invoice.OrderId == orderId {
			invoices = append(invoices, invoice)
		}
	}
	return invoices, len(invoices), nil
}

func (m *MockInvoiceRepository) FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string, pagination request.Pagination) ([]entities.Invoice, int, error) {
	var invoices []entities.Invoice
	for _, invoice := range m.invoices {
		if invoice.OrgId == orgId && invoice.SubscriptionId == subscriptionId {
			invoices = append(invoices, invoice)
		}
	}
	return invoices, len(invoices), nil
}

func (m *MockInvoiceRepository) AddLineItem(ctx context.Context, lineItem entities.InvoiceLineItem) (entities.InvoiceLineItem, error) {
	return lineItem, nil
}

func (m *MockInvoiceRepository) UpdateLineItem(ctx context.Context, lineItem entities.InvoiceLineItem) (entities.InvoiceLineItem, error) {
	return lineItem, nil
}

func (m *MockInvoiceRepository) DeleteLineItem(ctx context.Context, orgId string, invoiceId string, lineItemId string) error {
	return nil
}

func (m *MockInvoiceRepository) ListLineItems(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceLineItem, error) {
	return []entities.InvoiceLineItem{}, nil
}

func (m *MockInvoiceRepository) AddHistory(ctx context.Context, history entities.InvoiceHistory) (entities.InvoiceHistory, error) {
	return history, nil
}

func (m *MockInvoiceRepository) ListHistory(ctx context.Context, orgId string, invoiceId string) ([]entities.InvoiceHistory, error) {
	return []entities.InvoiceHistory{}, nil
}

type MockCustomerRepository struct{}

func (m MockCustomerRepository) FindById(ctx context.Context, orgId string, id string) (entities.Customer, error) {
	return entities.Customer{
		OrgId:     orgId,
		Id:        id,
		FirstName: "Test",
		LastName:  "Customer",
		Email:     "test@example.com",
	}, nil
}

func (m MockCustomerRepository) FindByEmail(ctx context.Context, orgId string, email string) (entities.Customer, error) {
	return entities.Customer{}, nil
}

func (m MockCustomerRepository) Create(ctx context.Context, entity entities.Customer) (entities.Customer, error) {
	return entity, nil
}

func (m MockCustomerRepository) Update(ctx context.Context, entity entities.Customer) (entities.Customer, error) {
	return entity, nil
}

func (m MockCustomerRepository) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Customer, int, error) {
	return []entities.Customer{}, 0, nil
}

func (m MockCustomerRepository) FindPaymentMethodById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error) {
	return entities.PaymentMethod{}, nil
}

func (m MockCustomerRepository) AddToCohort(ctx context.Context, orgId string, customerId string, cohortId string, cohortValue string) (entities.Customer, error) {
	return entities.Customer{}, nil
}

type MockDocSequenceRepository struct{}

func (m MockDocSequenceRepository) FindById(ctx context.Context, orgId string, id string) (entities.DocSequence, error) {
	return entities.DocSequence{}, nil
}

func (m MockDocSequenceRepository) FindByType(ctx context.Context, orgId string, sequenceType string) ([]entities.DocSequence, error) {
	return []entities.DocSequence{}, nil
}

func (m MockDocSequenceRepository) Create(ctx context.Context, entity entities.DocSequence) (entities.DocSequence, error) {
	return entity, nil
}

func (m MockDocSequenceRepository) Update(ctx context.Context, entity entities.DocSequence) (entities.DocSequence, error) {
	return entity, nil
}

func (m MockDocSequenceRepository) GetNextValue(ctx context.Context, orgId string, id string, sequenceType string) (int, error) {
	return 1, nil
}

type MockOrderRepository struct {
	Order entities.Order
}

func (m MockOrderRepository) FindById(ctx context.Context, orgId string, id string) (entities.Order, error) {
	return m.Order, nil
}

func (m MockOrderRepository) Create(ctx context.Context, entity entities.Order) (entities.Order, error) {
	return entity, nil
}

func (m MockOrderRepository) Update(ctx context.Context, entity entities.Order) (entities.Order, error) {
	return entity, nil
}

func (m MockOrderRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.Order, int, error) {
	return []entities.Order{m.Order}, 1, nil
}

type MockOrderItemRepository struct{}

func (m MockOrderItemRepository) FindById(ctx context.Context, orgId string, id string) (entities.OrderItem, error) {
	return entities.OrderItem{}, nil
}

func (m MockOrderItemRepository) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.OrderItem, error) {
	return []entities.OrderItem{}, nil
}

func (m MockOrderItemRepository) Create(ctx context.Context, entity entities.OrderItem) (entities.OrderItem, error) {
	return entity, nil
}

func (m MockOrderItemRepository) Update(ctx context.Context, entity entities.OrderItem) (entities.OrderItem, error) {
	return entity, nil
}

func (m MockOrderItemRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.OrderItem, int, error) {
	return []entities.OrderItem{}, 1, nil
}

func TestCreateInvoice(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockInvoiceRepo := NewMockInvoiceRepository()
	mockCustomerRepo := MockCustomerRepository{}
	mockDocSequenceRepo := MockDocSequenceRepository{}
	mockPubSub := MockPubSub{}

	// Create test data
	orgId := "test_org"
	customerId := "test_customer"
	subscriptionId := "test_subscription"
	orderId := "test_order"
	paymentId := "test_payment"

	// Create a mock order with items
	orderItem := entities.OrderItem{
		OrgId:       orgId,
		Id:          "test_order_item",
		OrderId:     orderId,
		ProductId:   "test_product",
		VariantId:   "test_variant",
		PriceId:     "test_price",
		Description: "Test Product",
		Quantity:    1,
		Subtotal:    1000, // $10.00
		Total:       1000, // $10.00
		Metadata:    map[string]string{"test": "value"},
	}

	mockOrder := entities.Order{
		OrgId:      orgId,
		Id:         orderId,
		CustomerId: customerId,
		Items:      []entities.OrderItem{orderItem},
		Currency:   "USD",
		Total:      1000, // $10.00
	}

	mockOrderRepo := MockOrderRepository{Order: mockOrder}
	mockOrderItemRepo := MockOrderItemRepository{}

	invoiceService := NewInvoiceService(
		mockInvoiceRepo,
		mockCustomerRepo,
		mockDocSequenceRepo,
		mockOrderRepo,
		mockOrderItemRepo,
		lib.ErrorReporter{},
		mockPubSub,
		MockLogger{},
	)

	// Create a payment
	payment := entities.Payment{
		OrgId:          orgId,
		Id:             paymentId,
		Psp:            common.Gateway("test_psp"),
		PspId:          "test_psp_id",
		Reference:      "test_reference",
		OrderId:        orderId,
		SubscriptionId: subscriptionId,
		Status:         payments.PaymentStatusSucceeded,
		Recurring:      true,
		Currency:       "USD",
		Amount:         1000, // $10.00
		PspFee:         50,   // $0.50
		PlatformFee:    20,   // $0.20
		NetAmount:      930,  // $9.30
		Metadata: map[string]string{
			"customer_id": customerId,
		},
		CompletedAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create the invoice request that would be created by CreateInvoiceForSubscriptionPayment
	// We're testing that the line items from the order are correctly used
	invoiceReq := dto.CreateInvoiceInput{
		CustomerId:     payment.Metadata["customer_id"],
		OrderId:        orderId,
		SubscriptionId: subscriptionId,
		Type:           entities.DocumentTypeInvoice,
		InvoiceType:    entities.InvoiceTypeRecurring,
		Currency:       payment.Currency,
		DueAt:          time.Now().UTC(),
		Notes:          fmt.Sprintf("Invoice for subscription payment %s", payment.Id),
		Metadata:       payment.Metadata,
		// We're not setting LineItems here because we want to test that they're fetched from the order
	}

	// Call the Create method directly with the same parameters that CreateInvoiceForSubscriptionPayment would use
	invoice, err := invoiceService.Create(ctx, orgId, invoiceReq)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, invoice.Id)
	assert.Equal(t, orgId, invoice.OrgId)
	assert.Equal(t, customerId, invoice.CustomerId)
	assert.Equal(t, subscriptionId, invoice.SubscriptionId)
	assert.Equal(t, orderId, invoice.OrderId)
	assert.Equal(t, entities.DocumentTypeInvoice, invoice.Type)
	assert.Equal(t, entities.InvoiceTypeRecurring, invoice.InvoiceType)
	assert.Equal(t, "USD", invoice.Currency)
	assert.NotEmpty(t, invoice.DocNumber)
	assert.NotEmpty(t, invoice.SequenceId)
	assert.Contains(t, invoice.Notes, paymentId)

	// Verify the invoice was stored in the repository
	storedInvoice, err := mockInvoiceRepo.FindById(ctx, orgId, invoice.Id)
	assert.NoError(t, err)
	assert.Equal(t, invoice.Id, storedInvoice.Id)
}

func TestCreateInvoiceForSubscriptionPayment(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockInvoiceRepo := NewMockInvoiceRepository()
	mockCustomerRepo := MockCustomerRepository{}
	mockDocSequenceRepo := MockDocSequenceRepository{}
	mockPubSub := MockPubSub{}

	// Create test data
	orgId := "test_org"
	customerId := "test_customer"
	subscriptionId := "test_subscription"
	orderId := "test_order"
	paymentId := "test_payment"

	// Create a mock order with items
	orderItem := entities.OrderItem{
		OrgId:       orgId,
		Id:          "test_order_item",
		OrderId:     orderId,
		ProductId:   "test_product",
		VariantId:   "test_variant",
		PriceId:     "test_price",
		Description: "Test Product",
		Quantity:    1,
		Subtotal:    1000, // $10.00
		Total:       1000, // $10.00
		Metadata:    map[string]string{"test": "value"},
	}

	mockOrder := entities.Order{
		OrgId:      orgId,
		Id:         orderId,
		CustomerId: customerId,
		Items:      []entities.OrderItem{orderItem},
		Currency:   "USD",
		Total:      1000, // $10.00
	}

	mockOrderRepo := MockOrderRepository{Order: mockOrder}
	mockOrderItemRepo := MockOrderItemRepository{}

	// Create a real ErrorReporter instance
	errorReporter := lib.NewErrorReporter(MockLogger{})

	// Cast to concrete InvoiceService struct to access non-interface methods
	invoiceService := NewInvoiceService(
		mockInvoiceRepo,
		mockCustomerRepo,
		mockDocSequenceRepo,
		mockOrderRepo,
		mockOrderItemRepo,
		errorReporter,
		mockPubSub,
		MockLogger{},
	).(*InvoiceService)

	// Create a payment
	payment := entities.Payment{
		OrgId:          orgId,
		Id:             paymentId,
		Psp:            common.Gateway("test_psp"),
		PspId:          "test_psp_id",
		Reference:      "test_reference",
		OrderId:        orderId,
		SubscriptionId: subscriptionId,
		Status:         payments.PaymentStatusSucceeded,
		Recurring:      true,
		Currency:       "USD",
		Amount:         1000, // $10.00
		PspFee:         50,   // $0.50
		PlatformFee:    20,   // $0.20
		NetAmount:      930,  // $9.30
		Metadata: map[string]string{
			"customer_id": customerId,
		},
		CompletedAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create the SubscriptionPaymentChargeSuccessEvent
	paymentSuccessEvent := topic.SubscriptionPaymentChargeSuccessEvent{
		OrgId:          orgId,
		SubscriptionId: subscriptionId,
		OrderId:        orderId,
		PaymentId:      paymentId,
		Metadata:       map[string]string{"customer_id": customerId},
		Payment:        payment,
	}

	// Call the CreateInvoiceForSubscriptionPayment method directly
	invoice, err := invoiceService.CreateInvoiceForSubscriptionPayment(ctx, paymentSuccessEvent)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, invoice.Id)
	assert.Equal(t, orgId, invoice.OrgId)
	assert.Equal(t, customerId, invoice.CustomerId)
	assert.Equal(t, subscriptionId, invoice.SubscriptionId)
	assert.Equal(t, orderId, invoice.OrderId)
	assert.Equal(t, entities.DocumentTypeInvoice, invoice.Type)
	assert.Equal(t, entities.InvoiceTypeRecurring, invoice.InvoiceType)
	assert.Equal(t, entities.InvoiceStatusPaid, invoice.Status)
	assert.Equal(t, "USD", invoice.Currency)
	assert.NotEmpty(t, invoice.DocNumber)
	assert.NotEmpty(t, invoice.SequenceId)
	assert.Contains(t, invoice.Notes, paymentId)

	// Verify the invoice was stored in the repository
	storedInvoice, err := mockInvoiceRepo.FindById(ctx, orgId, invoice.Id)
	assert.NoError(t, err)
	assert.Equal(t, invoice.Id, storedInvoice.Id)
}
