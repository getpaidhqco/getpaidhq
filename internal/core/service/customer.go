package service

import (
	"context"
	"encoding/json"
	"errors"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"time"
)

type CustomerService struct {
	customerRepository      port.CustomerRepository
	paymentMethodRepository port.PaymentMethodRepository
	pubsub                  port.PubSub
	logger                  port.Logger
}

// NewCustomerService schedules the monthly payment-method-expiry job
// and subscribes to order events. Both startup steps now return
// errors instead of panicking — a flaky cron or NATS during boot used
// to crash the process before the HTTP server came up.
func NewCustomerService(
	customerRepository port.CustomerRepository,
	paymentMethodRepository port.PaymentMethodRepository,
	pubsub port.PubSub,
	logger port.Logger,
	scheduler port.Scheduler,
) (*CustomerService, error) {
	svc := &CustomerService{
		customerRepository:      customerRepository,
		paymentMethodRepository: paymentMethodRepository,
		pubsub:                  pubsub,
		logger:                  logger,
	}
	// 3am first of every month — payment method expiry detection.
	if err := scheduler.ScheduleTask("0 3 1 * *", svc.DetectExpiringPaymentMethods); err != nil {
		return nil, err
	}
	// Order events feed cohort tracking. A subscribe failure isn't
	// fatal at boot in the same way — but surfacing the error lets
	// the caller decide.
	if _, err := pubsub.Subscribe(port.TopicOrderCompleted, safePubSubHandler(logger, "CustomerService.HandleOrderEvent", svc.HandleOrderEvent)); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *CustomerService) Create(ctx context.Context, orgId string, input CreateCustomerInput) (domain.Customer, error) {
	// check for existing customer — a not-found result is the expected happy
	// path (no duplicate), so only a real lookup failure should abort.
	exists, err := s.customerRepository.FindByEmail(ctx, orgId, input.Email)
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return domain.Customer{}, lib.NewCustomError(lib.InternalError, "Error creating customer", err)
	}
	if exists.Id != "" {
		return domain.Customer{}, lib.NewCustomError(lib.BadRequestError, "Customer already exists", nil)
	}

	customer := domain.Customer{
		OrgId:          orgId,
		Id:             lib.GenerateId("cus"),
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Email:          input.Email,
		Phone:          input.Phone,
		BillingAddress: input.BillingAddress,
		Metadata:       input.Metadata,
		CreatedAt:      time.Time{},
		UpdatedAt:      time.Time{},
	}

	newCustomer, err := s.customerRepository.Create(ctx, customer)
	if err != nil {
		s.logger.Error("Failed to create customer: ", err)
		return domain.Customer{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicCustomerCreated, newCustomer)
	return newCustomer, nil
}

func (s *CustomerService) GetPaymentMethod(ctx context.Context, orgId string, id string) (domain.PaymentMethod, error) {
	paymentMethod, err := s.paymentMethodRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get payment method: ", err)
		return domain.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
	}

	return paymentMethod, nil
}

func (s *CustomerService) CreatePaymentMethod(ctx context.Context, orgId string, input CreatePaymentMethodInput) (domain.PaymentMethod, error) {

	customer, err := s.customerRepository.FindById(ctx, orgId, input.CustomerId)
	if err != nil {
		s.logger.Error("Failed to get customer: ", err)
		return domain.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	var billingAddress = customer.BillingAddress
	if !input.BillingAddress.IsEmpty() {
		billingAddress = input.BillingAddress
	}
	if billingAddress.IsEmpty() {
		return domain.PaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Either specify billing address or add a default billing address to the customer.", nil)
	}

	var expireAt time.Time
	if input.Details != nil {
		details, err := domain.ParsePaymentMethodDetails(input.Type, input.Details)
		if err != nil {
			return domain.PaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Invalid card details", err)
		}

		expireAt = details.GetExpiryDate()
		s.logger.Debugf("This payment method expires at: %v", expireAt)
	}

	// check for existing payment method
	paymentMethod := domain.PaymentMethod{
		OrgId:          orgId,
		Id:             lib.GenerateId("pm"),
		Psp:            input.Psp,
		Name:           input.Name,
		Status:         domain.PaymentMethodStatusActive,
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
		return domain.PaymentMethod{}, lib.NewCustomError(lib.InternalError, "An internal error occurred", err)
	}

	if input.IsDefault {
		// update the customer's default payment method
		s.logger.Debugf("Updating customer %s default payment method to %s", customer.Id, newPaymentMethod.Id)
		customer.DefaultPaymentMethodId = newPaymentMethod.Id
		_, err = s.customerRepository.Update(ctx, customer)
		if err != nil {
			s.logger.Error("Failed to update customer: ", "err", err)
			return domain.PaymentMethod{}, lib.NewCustomError(lib.InternalError, "An internal error occurred", err)
		}
	}

	_ = s.pubsub.Publish(orgId, port.TopicPaymentMethodCreated, newPaymentMethod)
	return newPaymentMethod, nil
}

func (s *CustomerService) UpdatePaymentMethod(ctx context.Context, orgId string, input UpdatePaymentMethodInput) (domain.PaymentMethod, error) {

	customer, err := s.customerRepository.FindById(ctx, orgId, input.CustomerId)
	if err != nil {
		s.logger.Error("Failed to get customer: ", err)
		return domain.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	paymentMethod, err := s.paymentMethodRepository.FindById(ctx, orgId, input.PaymentMethodId)
	if err != nil {
		return domain.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
	}

	if !input.BillingAddress.IsEmpty() {
		paymentMethod.BillingAddress = input.BillingAddress
	}

	if input.Details != nil {
		details, err := domain.ParsePaymentMethodDetails(input.Type, input.Details)
		if err != nil {
			return domain.PaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Invalid card details", err)
		}

		paymentMethod.ExpireAt = details.GetExpiryDate()
		s.logger.Debugf("This payment method expires at: %v", paymentMethod.ExpireAt)
	}

	if input.Token != "" {
		paymentMethod.Token = input.Token
	}
	if input.Details != nil {
		paymentMethod.Details = input.Details
	}

	newPaymentMethod, err := s.paymentMethodRepository.Update(ctx, paymentMethod)
	if err != nil {
		s.logger.Error("Failed to update payment method: ", "err", err)
		return domain.PaymentMethod{}, lib.NewCustomError(lib.InternalError, "An internal error occurred", err)
	}

	if input.IsDefault {
		// update the customer's default payment method
		s.logger.Debugf("Updating customer %s default payment method to %s", customer.Id, newPaymentMethod.Id)
		customer.DefaultPaymentMethodId = newPaymentMethod.Id
		_, err = s.customerRepository.Update(ctx, customer)
		if err != nil {
			s.logger.Error("Failed to update customer: ", "err", err)
			return domain.PaymentMethod{}, lib.NewCustomError(lib.InternalError, "An internal error occurred", err)
		}
	}

	_ = s.pubsub.Publish(orgId, port.TopicPaymentMethodUpdated, newPaymentMethod)
	return newPaymentMethod, nil
}

func (s *CustomerService) DetectExpiringPaymentMethods() {
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
		_ = s.pubsub.Publish(paymentMethod.OrgId, port.TopicPaymentMethodExpired, paymentMethod)
	}
}

func (s *CustomerService) HandleOrderEvent(eventTopic string, data []byte) {

	var payload port.PubSubPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal payload: %v", err)
		return
	}

	switch eventTopic {
	case port.TopicOrderCompleted:
		var order domain.Order
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

func (s *CustomerService) Get(ctx context.Context, orgId string, id string) (domain.Customer, error) {
	customer, err := s.customerRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get customer: ", err)
		return domain.Customer{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	return customer, nil
}

func (s *CustomerService) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.Customer, int, error) {
	customers, total, err := s.customerRepository.List(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list customers: ", err)
		return nil, 0, lib.NewCustomError(lib.InternalError, "Error listing customers", err)
	}

	return customers, total, nil
}
