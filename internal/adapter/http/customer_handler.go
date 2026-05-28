package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
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
	fuego.Get(g, "", cc.List, option.Summary("List customers"))
	fuego.Get(g, "/{id}", cc.Get, option.Summary("Get a customer"))
	fuego.Post(g, "", cc.Create, option.Summary("Create a customer"))
	fuego.Post(g, "/{id}/payment-methods", cc.CreateCustomerPaymentMethod, option.Summary("Add a payment method to a customer"))
	fuego.Put(g, "/{id}/payment-methods/{pmid}", cc.UpdateCustomerPaymentMethod, option.Summary("Update a customer's payment method"))
}

func (cc *CustomerHandler) Create(c fuego.ContextWithBody[domain.CreateCustomerInput]) (domain.Customer, error) {
	authUser := AuthUserFrom(c)
	if !cc.authz.Enforce(authUser, port.ActionCreateCustomer, "") {
		return domain.Customer{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return domain.Customer{}, err
	}

	customer, err := cc.customerService.Create(c.Context(), authUser.OrgId, input)
	if err != nil {
		return domain.Customer{}, NewApiErrorFromError(err)
	}
	return customer, nil
}

func (cc *CustomerHandler) CreateCustomerPaymentMethod(c fuego.ContextWithBody[domain.CreatePaymentMethodInput]) (domain.PaymentMethod, error) {
	authUser := AuthUserFrom(c)
	if !cc.authz.Enforce(authUser, port.ActionCreatePaymentMethod, "") {
		return domain.PaymentMethod{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return domain.PaymentMethod{}, err
	}
	input.OrgId = authUser.OrgId
	input.CustomerId = c.PathParam("id")

	pm, err := cc.customerService.CreatePaymentMethod(c.Context(), authUser.OrgId, input)
	if err != nil {
		return domain.PaymentMethod{}, NewApiErrorFromError(err)
	}
	return pm, nil
}

func (cc *CustomerHandler) UpdateCustomerPaymentMethod(c fuego.ContextWithBody[domain.UpdatePaymentMethodInput]) (domain.PaymentMethod, error) {
	authUser := AuthUserFrom(c)
	if !cc.authz.Enforce(authUser, port.ActionUpdatePaymentMethod, "") {
		return domain.PaymentMethod{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return domain.PaymentMethod{}, err
	}
	input.OrgId = authUser.OrgId
	input.CustomerId = c.PathParam("id")
	input.PaymentMethodId = c.PathParam("pmid")

	pm, err := cc.customerService.UpdatePaymentMethod(c.Context(), authUser.OrgId, input)
	if err != nil {
		return domain.PaymentMethod{}, NewApiErrorFromError(err)
	}
	return pm, nil
}

func (cc *CustomerHandler) GetCustomerPaymentMethod(c fuego.ContextNoBody) (domain.PaymentMethod, error) {
	authUser := AuthUserFrom(c)
	pm, err := cc.customerService.GetPaymentMethod(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return domain.PaymentMethod{}, NewApiErrorFromError(err)
	}
	return pm, nil
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
