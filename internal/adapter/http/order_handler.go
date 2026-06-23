package handler

import (
	"time"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// OrderHandler handles HTTP requests for orders.
type OrderHandler struct {
	service *service.OrderService
	logger  port.Logger
	authz   port.Authz
}

func NewOrderHandler(orderService *service.OrderService, logger port.Logger, authz port.Authz) *OrderHandler {
	return &OrderHandler{
		service: orderService,
		logger:  logger,
		authz:   authz,
	}
}

func (o *OrderHandler) RegisterRoutes(s *fuego.Server) {
	g := fuego.Group(s, "/orders", option.Tags("Orders"))
	fuego.Post(g, "", o.CreateOrder, option.Summary("Create an order"), option.OperationID("createOrder"))
	fuego.Post(g, "/{id}/complete", o.CompleteOrder, option.Summary("Complete an order"), option.OperationID("completeOrder"))
	fuego.Get(g, "/{id}", o.Get, option.Summary("Get an order"), option.OperationID("getOrder"))
	fuego.Get(g, "", o.List, append(PaginationParams(), option.Summary("List orders"), option.OperationID("listOrders"))...)
	fuego.Get(g, "/{id}/subscriptions", o.ListSubscriptions, option.Summary("List subscriptions for an order"), option.OperationID("listOrderSubscriptions"))
}

type CreateOrderResponse struct {
	Order OrderResponse `json:"order"`
	Psp   any           `json:"psp"`
}

func (o *OrderHandler) CreateOrder(c fuego.ContextWithBody[CreateOrderRequest]) (CreateOrderResponse, error) {
	authUser := AuthUserFrom(c)
	if !o.authz.Enforce(authUser, port.ActionCreateOrder, "") {
		return CreateOrderResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}

	input, err := c.Body()
	if err != nil {
		return CreateOrderResponse{}, err
	}

	if input.SessionId == "" && len(input.Cart.Items) == 0 {
		return CreateOrderResponse{}, NewApiError(lib.ValidationError, "You must specify cart or session_id", nil)
	}
	if len(input.Cart.Items) > 0 && input.Cart.Currency == "" {
		return CreateOrderResponse{}, NewApiError(lib.ValidationError, "Currency is required", nil)
	}

	rsp, err := o.service.CreateOrder(c.Context(), port.CreateOrderInput{
		OrgId:    authUser.OrgId,
		Currency: input.Cart.Currency,
		Customer: domain.CreateOrderCommandCustomer{
			Id:        input.Customer.ID,
			Email:     input.Customer.Email,
			FirstName: input.Customer.FirstName,
			LastName:  input.Customer.LastName,
			Phone:     input.Customer.Phone,
			Metadata:  nil,
		},
		SessionId:       input.SessionId,
		PaymentMethodId: input.PaymentMethodId,
		CartItems:       ToCartItems(input.Cart.Items),
		PspId:           domain.Gateway(input.PspId),
		Metadata:        nil,
		Options:         input.Options,
	})
	if err != nil {
		return CreateOrderResponse{}, NewApiErrorFromError(err)
	}

	details, err := o.service.GetDetails(c.Context(), authUser.OrgId, rsp.Order.Id)
	if err != nil {
		return CreateOrderResponse{}, NewApiErrorFromError(err)
	}
	return CreateOrderResponse{
		Order: NewOrderResponseFromDetails(details),
		Psp:   rsp.Psp.PspResponse,
	}, nil
}

func (o *OrderHandler) CompleteOrder(c fuego.ContextWithBody[CompleteOrderRequest]) (OrderResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")

	if !o.authz.Enforce(authUser, port.ActionCreateOrder, "") {
		return OrderResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}

	input, err := c.Body()
	if err != nil {
		return OrderResponse{}, err
	}

	var completedAt time.Time
	if input.Payment.CompletedAt != "" {
		parsed, perr := time.Parse(time.RFC3339, input.Payment.CompletedAt)
		if perr != nil {
			return OrderResponse{}, NewApiError(lib.ValidationError, "Invalid completed_at format", nil)
		}
		completedAt = parsed
	}

	order, err := o.service.CompleteOrder(c.Context(), port.CompleteOrderInput{
		OrgId:           authUser.OrgId,
		Id:              id,
		PaymentMethodId: input.PaymentMethodId,
		PaymentMethod: port.CompleteOrderInputPaymentMethod{
			Psp:       input.PaymentMethod.Psp,
			Name:      input.PaymentMethod.Name,
			IsDefault: input.PaymentMethod.IsDefault,
			Details:   input.PaymentMethod.Details,
			BillingAddress: domain.Address{
				FirstName:  input.PaymentMethod.BillingAddress.FirstName,
				LastName:   input.PaymentMethod.BillingAddress.LastName,
				Email:      input.PaymentMethod.BillingAddress.Email,
				Phone:      input.PaymentMethod.BillingAddress.Phone,
				Line1:      input.PaymentMethod.BillingAddress.Line1,
				Line2:      input.PaymentMethod.BillingAddress.Line2,
				City:       input.PaymentMethod.BillingAddress.City,
				State:      input.PaymentMethod.BillingAddress.State,
				PostalCode: input.PaymentMethod.BillingAddress.PostalCode,
				Country:    domain.Country(input.PaymentMethod.BillingAddress.Country),
			},
			Type:     domain.PaymentMethodType(input.PaymentMethod.Type),
			Token:    input.PaymentMethod.Token,
			Metadata: input.PaymentMethod.Metadata,
		},
		Payment: port.CompleteOrderInputPayment{
			PspId:       input.Payment.PspId,
			CompletedAt: completedAt,
			Reference:   input.Payment.Reference,
			Amount:      input.Payment.Amount,
			Currency:    input.Payment.Currency,
			Metadata:    input.Payment.Metadata,
		},
		Metadata: input.Metadata,
	})
	if err != nil {
		return OrderResponse{}, NewApiErrorFromError(err)
	}
	details, err := o.service.GetDetails(c.Context(), authUser.OrgId, order.Id)
	if err != nil {
		return OrderResponse{}, NewApiErrorFromError(err)
	}
	return NewOrderResponseFromDetails(details), nil
}

func (o *OrderHandler) ListSubscriptions(c fuego.ContextNoBody) ([]SubscriptionResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")

	if !o.authz.Enforce(authUser, port.ActionListOrderSubscriptions, "") {
		return nil, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}

	subs, err := o.service.ListOrderSubscriptions(c.Context(), authUser.OrgId, id)
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}

	// Map through the snake_case DTO — returning raw domain.Subscription leaked
	// PascalCase JSON, unlike every other subscription endpoint. All of an
	// order's subscriptions share the order's customer, so fetch it once.
	details, err := o.service.GetDetails(c.Context(), authUser.OrgId, id)
	if err != nil {
		return nil, NewApiErrorFromError(err)
	}
	rsp := make([]SubscriptionResponse, len(subs))
	for i, sub := range subs {
		rsp[i] = NewSubscriptionResponseFromDetails(service.SubscriptionDetails{
			Subscription: sub,
			Customer:     details.Customer,
		})
	}
	return rsp, nil
}

func (o *OrderHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	authUser := AuthUserFrom(c)
	pagination := GetPagination(c)

	details, total, err := o.service.ListDetails(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	orderRsp := make([]OrderResponse, 0, len(details))
	for _, d := range details {
		orderRsp = append(orderRsp, NewOrderResponseFromDetails(d))
	}

	return ListResponse{
		Data: orderRsp,
		Meta: Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	}, nil
}

func (o *OrderHandler) Get(c fuego.ContextNoBody) (OrderResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")

	details, err := o.service.GetDetails(c.Context(), authUser.OrgId, id)
	if err != nil {
		return OrderResponse{}, NewApiErrorFromError(err)
	}

	return NewOrderResponseFromDetails(details), nil
}
