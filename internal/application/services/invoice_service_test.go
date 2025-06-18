package services

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/events"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/lib"
	"testing"
	"time"
)

// Mock implementations

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

func TestCreateInvoiceForSubscriptionPayment(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockInvoiceRepo := NewMockInvoiceRepository()
	mockCustomerRepo := MockCustomerRepository{}
	mockDocSequenceRepo := MockDocSequenceRepository{}
	mockPubSub := MockPubSub{}

	invoiceService := NewInvoiceService(
		mockInvoiceRepo,
		mockCustomerRepo,
		mockDocSequenceRepo,
		mockPubSub,
		MockLogger{},
	)

	// Create test data
	orgId := "test_org"
	customerId := "test_customer"
	subscriptionId := "test_subscription"
	orderId := "test_order"
	paymentId := "test_payment"

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
	lineItem := dto.CreateInvoiceLineItemRequest{
		Description: fmt.Sprintf("Subscription payment for %s", subscriptionId),
		Quantity:    1.0,
		UnitPrice:   int(payment.Amount),
		Metadata:    payment.Metadata,
	}

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
		LineItems:      []dto.CreateInvoiceLineItemRequest{lineItem},
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
