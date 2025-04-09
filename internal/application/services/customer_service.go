package services

import (
	"context"
	"encoding/json"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payment_methods"
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
) interfaces.CustomerService {
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

	// subscribe to order events to manage cohorts
	_, err = pubsub.Subscribe(topic.OrderCompleted, service.HandleOrderEvent)

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

	_ = s.pubsub.Publish(orgId, topic.CustomerCreated, newCustomer)
	return newCustomer, nil
}

func (s CustomerService) GetPaymentMethod(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error) {
	paymentMethod, err := s.paymentMethodRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get payment method: ", err)
		return entities.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
	}

	return paymentMethod, nil
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

	var expireAt time.Time
	if input.Details != "" {
		details, err := payment_methods.ParseDetails(input.Type, input.Details)
		if err != nil {
			return entities.PaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Invalid card details", err)
		}

		expireAt = details.GetExpiryDate()
		s.logger.Debugf("This payment method expires at: %v", expireAt)
	}

	// check for existing payment method
	paymentMethod := entities.PaymentMethod{
		OrgId:          orgId,
		Id:             lib.GenerateId("pm"),
		Psp:            input.Psp,
		Name:           input.Name,
		Status:         payment_methods.Active,
		CustomerId:     input.CustomerId,
		BillingAddress: billingAddress,
		Type:           input.Type,
		Token:          input.Token,
		Details:        input.Details,
		Metadata:       input.Metadata,
		ExpireAt:       expireAt,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	newPaymentMethod, err := s.paymentMethodRepository.Create(ctx, paymentMethod)
	if err != nil {
		s.logger.Error("Failed to create payment method: ", "err", err)
		return entities.PaymentMethod{}, lib.MapDatabaseError(err)
	}

	if input.IsDefault {
		// update the customer's default payment method
		s.logger.Debugf("Updating customer %s default payment method to %s", customer.Id, newPaymentMethod.Id)
		customer.DefaultPaymentMethodId = newPaymentMethod.Id
		_, err = s.customerRepository.Update(ctx, customer)
		if err != nil {
			s.logger.Error("Failed to update customer: ", "err", err)
			return entities.PaymentMethod{}, lib.MapDatabaseError(err)
		}
	}

	_ = s.pubsub.Publish(orgId, topic.PaymentMethodCreated, newPaymentMethod)
	return newPaymentMethod, nil
}

func (s CustomerService) UpdatePaymentMethod(ctx context.Context, orgId string, input interfaces.UpdatePaymentMethodInput) (entities.PaymentMethod, error) {

	customer, err := s.customerRepository.FindById(ctx, orgId, input.CustomerId)
	if err != nil {
		s.logger.Error("Failed to get customer: ", err)
		return entities.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	paymentMethod, err := s.paymentMethodRepository.FindById(ctx, orgId, input.PaymentMethodId)
	if err != nil {
		return entities.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
	}

	if !input.BillingAddress.IsEmpty() {
		paymentMethod.BillingAddress = input.BillingAddress
	}

	if input.Details != "" {
		details, err := payment_methods.ParseDetails(input.Type, input.Details)
		if err != nil {
			return entities.PaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Invalid card details", err)
		}

		paymentMethod.ExpireAt = details.GetExpiryDate()
		s.logger.Debugf("This payment method expires at: %v", paymentMethod.ExpireAt)
	}

	if input.Token != "" {
		paymentMethod.Token = input.Token
	}
	if input.Details != "" {
		paymentMethod.Details = input.Details
	}

	newPaymentMethod, err := s.paymentMethodRepository.Update(ctx, paymentMethod)
	if err != nil {
		s.logger.Error("Failed to update payment method: ", "err", err)
		return entities.PaymentMethod{}, lib.MapDatabaseError(err)
	}

	if input.IsDefault {
		// update the customer's default payment method
		s.logger.Debugf("Updating customer %s default payment method to %s", customer.Id, newPaymentMethod.Id)
		customer.DefaultPaymentMethodId = newPaymentMethod.Id
		_, err = s.customerRepository.Update(ctx, customer)
		if err != nil {
			s.logger.Error("Failed to update customer: ", "err", err)
			return entities.PaymentMethod{}, lib.MapDatabaseError(err)
		}
	}

	_ = s.pubsub.Publish(orgId, topic.PaymentMethodUpdated, newPaymentMethod)
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

func (s CustomerService) HandleOrderEvent(eventTopic string, data []byte) {

	var payload events.Payload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal payload: %v", err)
		return
	}

	switch eventTopic {
	case topic.OrderCompleted:
		var order entities.Order
		payloadBytes, err := json.Marshal(payload.Data)
		if err != nil {
			s.logger.Errorf("Failed to marshal payload data: %v", err)
			return
		}
		err = json.Unmarshal(payloadBytes, &order)
		if err != nil {
			s.logger.Errorf("Failed to unmarshal event data: %v", err)
			return
		}
		// add the customer to the signup_date cohort
		s.logger.Infof("Adding customer [%s] to the [signup_date] cohort", order.CustomerId)
		_, err = s.customerRepository.AddToCohort(
			context.Background(),
			order.OrgId,
			order.CustomerId,
			"signup_date",
			time.Now().Format("2006-01-02"),
		)
		if err != nil {
			s.logger.Errorf("Failed to add customer to cohort: %v", err)
			return
		}
	}
}
