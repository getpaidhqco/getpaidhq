package services

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type CustomerService struct {
	customerRepository      repositories.CustomerRepository
	paymentMethodRepository repositories.PaymentMethodRepository
	pubsub                  events.PubSub
	logger                  logger.Logger
}

func NewCustomerService(
	customerRepository repositories.CustomerRepository,
	paymentMethodRepository repositories.PaymentMethodRepository,
	pubsub events.PubSub,
	logger logger.Logger,
	scheduler interfaces.Scheduler,
) CustomerService {
	service := CustomerService{
		customerRepository:      customerRepository,
		paymentMethodRepository: paymentMethodRepository,
		pubsub:                  pubsub,
		logger:                  logger,
	}
	// set up the payment method expiry detection
	// 3am first of every month
	err := scheduler.ScheduleTask("0 3 1 * *", service.DetectExpiringPaymentMethods)
	if err != nil {
		logger.Errorf("Failed to schedule task: %v", err)
		panic(err)
	}
	return service
}

func (s CustomerService) Create(ctx context.Context, orgId string, customerRequest request.CreateCustomerRequest) (entities.Customer, error) {

	// check for existing customer
	exists, err := s.customerRepository.FindByEmail(ctx, orgId, customerRequest.Email)
	if err != nil {
		return entities.Customer{}, lib.NewCustomError(lib.InternalError, "Error creating customer", err)
	}
	if exists.Id != "" {
		return entities.Customer{}, lib.NewCustomError(lib.BadRequestError, "Customer already exists", nil)
	}

	customer := entities.Customer{
		OrgId:          orgId,
		Id:             lib.GenerateId("cus"),
		FirstName:      customerRequest.FirstName,
		LastName:       customerRequest.LastName,
		Email:          customerRequest.Email,
		Phone:          customerRequest.Phone,
		BillingAddress: customerRequest.BillingAddress,
		Metadata:       customerRequest.Metadata,
		CreatedAt:      time.Time{},
		UpdatedAt:      time.Time{},
	}

	newCustomer, err := s.customerRepository.Create(ctx, customer)
	if err != nil {
		s.logger.Error("Failed to create customer: ", err)
		return entities.Customer{}, err
	}

	return newCustomer, nil
}

func (s CustomerService) CreatePaymentMethod(ctx context.Context, orgId string, input interfaces.CreatePaymentMethodInput) (entities.PaymentMethod, error) {

	customer, err := s.customerRepository.FindById(ctx, orgId, input.CustomerId)
	if err != nil {
		s.logger.Error("Failed to get customer: ", err)
		return entities.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	var billingAddress = customer.BillingAddress
	if !input.BillingAddress.IsEmpty() {
		billingAddress = input.BillingAddress
	}
	if billingAddress.IsEmpty() {
		return entities.PaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Either specify billing address or add a default billing address to the customer.", nil)
	}

	// check for existing payment method
	paymentMethod := entities.PaymentMethod{
		OrgId:          orgId,
		Id:             lib.GenerateId("pm"),
		Psp:            input.Psp,
		Name:           input.Name,
		CustomerId:     input.CustomerId,
		IsDefault:      input.IsDefault,
		BillingAddress: billingAddress,
		Type:           input.Type,
		Token:          input.Token,
		Details:        input.Metadata,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	newPaymentMethod, err := s.customerRepository.CreatePaymentMethod(ctx, paymentMethod)
	if err != nil {
		s.logger.Error("Failed to create payment method: ", "err", err)
		return entities.PaymentMethod{}, lib.MapDatabaseError(err)
	}

	_ = s.pubsub.Publish(orgId, topic.PaymentMethodCreated, newPaymentMethod)
	return newPaymentMethod, nil
}

func (s CustomerService) DetectExpiringPaymentMethods() {
	s.logger.Infof("Detecting expiring payment methods for all organizations")
	// Implement the logic to detect expiring payment methods
	expiring, err := s.paymentMethodRepository.FindExpiringPaymentMethods(context.Background(), time.Now().UTC())
	if err != nil {
		s.logger.Error("Failed to detect expiring payment methods: ", "err", err)
		return
	}
	for _, paymentMethod := range expiring {
		// send notification to customer
		s.logger.Infof("Payment method %s is expiring", paymentMethod.Id)
		_ = s.pubsub.Publish(paymentMethod.OrgId, topic.PaymentMethodExpired, paymentMethod)
	}
}
