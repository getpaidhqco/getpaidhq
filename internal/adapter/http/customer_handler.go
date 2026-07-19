package handler

import (
	"getpaidhq/internal/lib/errors"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type CustomerHandler struct {
	customerService *service.CustomerService
	logger          port.Logger
	authz           port.Authz
}

func NewCustomerHandler(customerService *service.CustomerService, logger port.Logger, authz port.Authz) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
		logger:          logger,
		authz:           authz,
	}
}

func (cc *CustomerHandler) RegisterRoutes(s *fuego.Server) {
	g := fuego.Group(s, "/customers", option.Tags("Customers"))
	fuego.Get(g, "", cc.List, append(PaginationParams(), option.Summary("List customers"), option.OperationID("listCustomers"))...)
	fuego.Get(g, "/{id}", cc.Get, option.Summary("Get a customer"), option.OperationID("getCustomer"))
	fuego.Post(g, "", cc.Create, option.Summary("Create a customer"), option.OperationID("createCustomer"))
	fuego.Post(g, "/{id}/payment-methods", cc.CreateCustomerPaymentMethod, option.Summary("Add a payment method to a customer"), option.OperationID("createCustomerPaymentMethod"))
	fuego.Put(g, "/{id}/payment-methods/{pmid}", cc.UpdateCustomerPaymentMethod, option.Summary("Update a customer's payment method"), option.OperationID("updateCustomerPaymentMethod"))
}

func (cc *CustomerHandler) Create(c fuego.ContextWithBody[port.CreateCustomerInput]) (CustomerResponse, error) {
	authUser := AuthUserFrom(c)
	if !cc.authz.Enforce(authUser, port.ActionCreateCustomer, "") {
		return CustomerResponse{}, NewApiError(errors.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return CustomerResponse{}, err
	}

	customer, err := cc.customerService.Create(c.Context(), authUser.OrgId, input)
	if err != nil {
		return CustomerResponse{}, NewApiErrorFromError(err)
	}
	return NewCustomerFromEntity(customer), nil
}

func (cc *CustomerHandler) CreateCustomerPaymentMethod(c fuego.ContextWithBody[port.CreatePaymentMethodInput]) (PaymentMethodResponse, error) {
	authUser := AuthUserFrom(c)
	if !cc.authz.Enforce(authUser, port.ActionCreatePaymentMethod, "") {
		return PaymentMethodResponse{}, NewApiError(errors.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return PaymentMethodResponse{}, err
	}
	input.OrgId = authUser.OrgId
	input.CustomerId = c.PathParam("id")

	pm, err := cc.customerService.CreatePaymentMethod(c.Context(), authUser.OrgId, input)
	if err != nil {
		return PaymentMethodResponse{}, NewApiErrorFromError(err)
	}
	return NewPaymentMethodResponse(pm), nil
}

func (cc *CustomerHandler) UpdateCustomerPaymentMethod(c fuego.ContextWithBody[port.UpdatePaymentMethodInput]) (PaymentMethodResponse, error) {
	authUser := AuthUserFrom(c)
	if !cc.authz.Enforce(authUser, port.ActionUpdatePaymentMethod, "") {
		return PaymentMethodResponse{}, NewApiError(errors.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return PaymentMethodResponse{}, err
	}
	input.OrgId = authUser.OrgId
	input.CustomerId = c.PathParam("id")
	input.PaymentMethodId = c.PathParam("pmid")

	pm, err := cc.customerService.UpdatePaymentMethod(c.Context(), authUser.OrgId, input)
	if err != nil {
		return PaymentMethodResponse{}, NewApiErrorFromError(err)
	}
	return NewPaymentMethodResponse(pm), nil
}

func (cc *CustomerHandler) Get(c fuego.ContextNoBody) (CustomerResponse, error) {
	authUser := AuthUserFrom(c)
	customer, err := cc.customerService.Get(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return CustomerResponse{}, NewApiErrorFromError(err)
	}
	return NewCustomerFromEntity(customer), nil
}

func (cc *CustomerHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	authUser := AuthUserFrom(c)
	pagination := GetPagination(c)

	customers, total, err := cc.customerService.List(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	customerResponses := make([]CustomerResponse, len(customers))
	for i, customer := range customers {
		customerResponses[i] = NewCustomerFromEntity(customer)
	}
	return ListResponse{
		Data: customerResponses,
		Meta: Meta{Total: total, Page: pagination.Page, Limit: pagination.Limit},
	}, nil
}
