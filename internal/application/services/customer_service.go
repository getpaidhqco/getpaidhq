package services

import (
	"context"
	"encoding/json"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payment_methods"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/security"
	domainevents "payloop/internal/domain/events"
	"payloop/internal/lib"
	"time"
)

type CustomerService struct {
	customerRepository      repositories.CustomerRepository
	paymentMethodRepository repositories.PaymentMethodRepository
	tokenVault              security.TokenVault
	notificationPublisher   events.NotificationPublisher
	logger                  logger.Logger
}

func NewCustomerService(
	customerRepository repositories.CustomerRepository,
	paymentMethodRepository repositories.PaymentMethodRepository,
	tokenVault security.TokenVault,
	notificationPublisher events.NotificationPublisher,
	logger logger.Logger,
	scheduler interfaces.Scheduler,
) interfaces.CustomerService {
	service := CustomerService{
		customerRepository:      customerRepository,
		paymentMethodRepository: paymentMethodRepository,
		tokenVault:              tokenVault,
		notificationPublisher:   notificationPublisher,
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
	_, err = notificationPublisher.Subscribe(topic.OrderCompleted, service.HandleOrderEvent)

	return service
}

func (s CustomerService) Create(ctx context.Context, orgId string, input dto.CreateCustomerInput) (entities.Customer, error) {
	// check for existing customer
	exists, err := s.customerRepository.FindByEmail(ctx, orgId, input.Email)
	if err != nil {
		return entities.Customer{}, lib.NewCustomError(lib.InternalError, "Error creating customer", err)
	}
	if exists.Id != "" {
		return entities.Customer{}, lib.NewCustomError(lib.BadRequestError, "Customer already exists", nil)
	}

	customer := entities.Customer{
		OrgId:          orgId,
		Id:             lib.GenerateId("cus"),
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Email:          input.Email,
		Phone:          input.Phone,
		BillingAddress: *input.BillingAddress,
		Metadata:       input.Metadata,
		CreatedAt:      time.Time{},
		UpdatedAt:      time.Time{},
	}

	newCustomer, err := s.customerRepository.Create(ctx, customer)
	if err != nil {
		s.logger.Error("Failed to create customer: ", err)
		return entities.Customer{}, err
	}

	_ = s.notificationPublisher.Publish(orgId, topic.CustomerCreated, newCustomer)
	return newCustomer, nil
}

func (s CustomerService) Update(ctx context.Context, orgId string, customerId string, input dto.UpdateCustomerInput) (entities.Customer, error) {
	// Get existing customer
	customer, err := s.customerRepository.FindById(ctx, orgId, customerId)
	if err != nil {
		return entities.Customer{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	// Update fields if provided
	if input.Email != nil {
		customer.Email = *input.Email
	}
	if input.FirstName != nil {
		customer.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		customer.LastName = *input.LastName
	}
	if input.Phone != nil {
		customer.Phone = *input.Phone
	}
	if input.BillingAddress != nil {
		customer.BillingAddress = *input.BillingAddress
	}
	if input.Metadata != nil {
		customer.Metadata = input.Metadata
	}

	customer.UpdatedAt = time.Now().UTC()

	updatedCustomer, err := s.customerRepository.Update(ctx, customer)
	if err != nil {
		s.logger.Error("Failed to update customer: ", err)
		return entities.Customer{}, lib.MapDatabaseError(err)
	}

	// Note: Using CustomerCreated topic as CustomerUpdated is not defined
	_ = s.notificationPublisher.Publish(orgId, topic.CustomerCreated, updatedCustomer)
	return updatedCustomer, nil
}

func (s CustomerService) GetPaymentMethod(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error) {
	paymentMethod, err := s.paymentMethodRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get payment method: ", err)
		return entities.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
	}

	return paymentMethod, nil
}

func (s CustomerService) CreatePaymentMethod(ctx context.Context, orgId string, input dto.CreatePaymentMethodInput) (entities.PaymentMethod, error) {
	customer, err := s.customerRepository.FindById(ctx, orgId, input.CustomerId)
	if err != nil {
		s.logger.Error("Failed to get customer: ", err)
		return entities.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	var billingAddress = customer.BillingAddress
	if input.BillingAddress != nil && !input.BillingAddress.IsEmpty() {
		billingAddress = *input.BillingAddress
	}
	if billingAddress.IsEmpty() {
		return entities.PaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Either specify billing address or add a default billing address to the customer.", nil)
	}

	var expireAt time.Time
	if details, ok := input.Details.(string); ok && details != "" {
		parsedDetails, err := payment_methods.ParseDetails(input.Type, details)
		if err != nil {
			return entities.PaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Invalid card details", err)
		}

		expireAt = parsedDetails.GetExpiryDate()
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

	_ = s.notificationPublisher.Publish(orgId, topic.PaymentMethodCreated, newPaymentMethod)
	return newPaymentMethod, nil
}

func (s CustomerService) UpdatePaymentMethod(ctx context.Context, orgId string, paymentMethodId string, input dto.UpdatePaymentMethodInput) (entities.PaymentMethod, error) {
	paymentMethod, err := s.paymentMethodRepository.FindById(ctx, orgId, paymentMethodId)
	if err != nil {
		return entities.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
	}

	// Update fields if provided
	if input.Name != nil {
		paymentMethod.Name = *input.Name
	}
	if input.BillingAddress != nil {
		paymentMethod.BillingAddress = *input.BillingAddress
	}
	if input.Metadata != nil {
		paymentMethod.Metadata = input.Metadata
	}

	paymentMethod.UpdatedAt = time.Now().UTC()

	updatedPaymentMethod, err := s.paymentMethodRepository.Update(ctx, paymentMethod)
	if err != nil {
		s.logger.Error("Failed to update payment method: ", "err", err)
		return entities.PaymentMethod{}, lib.MapDatabaseError(err)
	}

	if input.IsDefault != nil && *input.IsDefault {
		// Get customer
		customer, err := s.customerRepository.FindById(ctx, orgId, paymentMethod.CustomerId)
		if err != nil {
			s.logger.Error("Failed to get customer: ", err)
			return entities.PaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
		}

		// update the customer's default payment method
		s.logger.Debugf("Updating customer %s default payment method to %s", customer.Id, updatedPaymentMethod.Id)
		customer.DefaultPaymentMethodId = updatedPaymentMethod.Id
		_, err = s.customerRepository.Update(ctx, customer)
		if err != nil {
			s.logger.Error("Failed to update customer: ", "err", err)
			return entities.PaymentMethod{}, lib.MapDatabaseError(err)
		}
	}

	_ = s.notificationPublisher.Publish(orgId, topic.PaymentMethodUpdated, updatedPaymentMethod)
	return updatedPaymentMethod, nil
}

// GetSecurePaymentMethod retrieves a payment method with secure token handling
func (s CustomerService) GetSecurePaymentMethod(ctx context.Context, orgId string, id string) (entities.SecurePaymentMethod, error) {
	securePaymentMethod, err := s.paymentMethodRepository.FindSecureById(ctx, orgId, id, s.tokenVault)
	if err != nil {
		s.logger.Error("Failed to get secure payment method: ", err)
		return entities.SecurePaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
	}

	return securePaymentMethod, nil
}

// CreateSecurePaymentMethod creates a payment method with encrypted token storage
func (s CustomerService) CreateSecurePaymentMethod(ctx context.Context, orgId string, input dto.CreatePaymentMethodInput) (entities.SecurePaymentMethod, error) {
	customer, err := s.customerRepository.FindById(ctx, orgId, input.CustomerId)
	if err != nil {
		s.logger.Error("Failed to get customer: ", err)
		return entities.SecurePaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	var billingAddress = customer.BillingAddress
	if input.BillingAddress != nil && !input.BillingAddress.IsEmpty() {
		billingAddress = *input.BillingAddress
	}
	if billingAddress.IsEmpty() {
		return entities.SecurePaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Either specify billing address or add a default billing address to the customer.", nil)
	}

	var expireAt time.Time
	if details, ok := input.Details.(string); ok && details != "" {
		parsedDetails, err := payment_methods.ParseDetails(input.Type, details)
		if err != nil {
			return entities.SecurePaymentMethod{}, lib.NewCustomError(lib.BadRequestError, "Invalid card details", err)
		}

		expireAt = parsedDetails.GetExpiryDate()
		s.logger.Debugf("This payment method expires at: %v", expireAt)
	}

	// Create payment method entity
	paymentMethod := entities.PaymentMethod{
		OrgId:          orgId,
		Id:             lib.GenerateId("pm"),
		CustomerId:     input.CustomerId,
		Psp:            input.Psp,
		Type:           input.Type,
		BillingAddress: billingAddress,
		Name:           input.Name,
		Details:        input.Details,
		Metadata:       input.Metadata,
		ExpireAt:       expireAt,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Create secure payment method using the security service
	securityService := entities.NewPaymentMethodSecurityService(s.tokenVault)
	securePaymentMethod, err := securityService.CreateSecurePaymentMethod(
		ctx,
		paymentMethod,
		input.Token,
	)
	if err != nil {
		s.logger.Error("Failed to create secure payment method: ", "err", err)
		return entities.SecurePaymentMethod{}, lib.NewCustomError(lib.InternalError, "Failed to create secure payment method", err)
	}

	// Save to database
	savedSecurePaymentMethod, err := s.paymentMethodRepository.CreateSecure(ctx, securePaymentMethod)
	if err != nil {
		s.logger.Error("Failed to save secure payment method: ", "err", err)
		return entities.SecurePaymentMethod{}, lib.MapDatabaseError(err)
	}

	if input.IsDefault {
		// update the customer's default payment method
		s.logger.Debugf("Updating customer %s default payment method to %s", customer.Id, savedSecurePaymentMethod.Id)
		customer.DefaultPaymentMethodId = savedSecurePaymentMethod.Id
		_, err = s.customerRepository.Update(ctx, customer)
		if err != nil {
			s.logger.Error("Failed to update customer: ", "err", err)
			return entities.SecurePaymentMethod{}, lib.MapDatabaseError(err)
		}
	}

	_ = s.notificationPublisher.Publish(orgId, topic.PaymentMethodCreated, savedSecurePaymentMethod.ToEntity())
	return savedSecurePaymentMethod, nil
}

// UpdateSecurePaymentMethod updates a payment method with encrypted token storage
func (s CustomerService) UpdateSecurePaymentMethod(ctx context.Context, orgId string, paymentMethodId string, input dto.UpdatePaymentMethodInput) (entities.SecurePaymentMethod, error) {
	securePaymentMethod, err := s.paymentMethodRepository.FindSecureById(ctx, orgId, paymentMethodId, s.tokenVault)
	if err != nil {
		return entities.SecurePaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
	}

	// Update fields if provided
	if input.Name != nil {
		securePaymentMethod.Name = *input.Name
	}
	if input.BillingAddress != nil {
		securePaymentMethod.BillingAddress = *input.BillingAddress
	}
	if input.Metadata != nil {
		securePaymentMethod.Metadata = input.Metadata
	}

	securePaymentMethod.UpdatedAt = time.Now().UTC()

	updatedSecurePaymentMethod, err := s.paymentMethodRepository.UpdateSecure(ctx, securePaymentMethod)
	if err != nil {
		s.logger.Error("Failed to update secure payment method: ", "err", err)
		return entities.SecurePaymentMethod{}, lib.MapDatabaseError(err)
	}

	if input.IsDefault != nil && *input.IsDefault {
		// Get customer
		customer, err := s.customerRepository.FindById(ctx, orgId, securePaymentMethod.CustomerId)
		if err != nil {
			s.logger.Error("Failed to get customer: ", err)
			return entities.SecurePaymentMethod{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
		}

		// update the customer's default payment method
		s.logger.Debugf("Updating customer %s default payment method to %s", customer.Id, updatedSecurePaymentMethod.Id)
		customer.DefaultPaymentMethodId = updatedSecurePaymentMethod.Id
		_, err = s.customerRepository.Update(ctx, customer)
		if err != nil {
			s.logger.Error("Failed to update customer: ", "err", err)
			return entities.SecurePaymentMethod{}, lib.MapDatabaseError(err)
		}
	}

	_ = s.notificationPublisher.Publish(orgId, topic.PaymentMethodUpdated, updatedSecurePaymentMethod.ToEntity())
	return updatedSecurePaymentMethod, nil
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
		_ = s.notificationPublisher.Publish(paymentMethod.OrgId, topic.PaymentMethodExpired, paymentMethod)
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
		var orderCompletedEvent domainevents.OrderCompletedEvent
		payloadBytes, err := json.Marshal(payload.Data)
		if err != nil {
			s.logger.Errorf("Failed to marshal payload data: %v", err)
			return
		}
		err = json.Unmarshal(payloadBytes, &orderCompletedEvent)
		if err != nil {
			s.logger.Errorf("Failed to unmarshal event data: %v", err)
			return
		}
		order := orderCompletedEvent.Order
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

func (s CustomerService) Get(ctx context.Context, orgId string, id string) (entities.Customer, error) {
	customer, err := s.customerRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to get customer: ", err)
		return entities.Customer{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	return customer, nil
}

func (s CustomerService) List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Customer], error) {
	// Convert application DTO pagination to request pagination
	requestPagination := request.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortDirection: pagination.SortDirection,
		SortBy:        pagination.SortBy,
	}

	customers, total, err := s.customerRepository.List(ctx, orgId, requestPagination)
	if err != nil {
		s.logger.Error("Failed to list customers: ", err)
		return dto.PaginatedResult[entities.Customer]{}, lib.NewCustomError(lib.InternalError, "Error listing customers", err)
	}

	result := dto.PaginatedResult[entities.Customer]{
		Items:      customers,
		TotalCount: total,
		Page:       pagination.Page,
		PageSize:   pagination.Limit,
	}

	return result, nil
}
