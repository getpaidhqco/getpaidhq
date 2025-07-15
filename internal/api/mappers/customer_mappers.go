package mappers

import (
    "payloop/internal/api/dto/request"
    "payloop/internal/api/dto/response"
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
)

// ToCreateCustomerInput converts API request to application input
func ToCreateCustomerInput(req request.CreateCustomerRequest) dto.CreateCustomerInput {
    // Create a pointer to the billing address
    var billingAddressPtr *entities.Address
    if (req.BillingAddress != entities.Address{}) {
        billingAddressPtr = &req.BillingAddress
    }

    return dto.CreateCustomerInput{
        Email:          req.Email,
        FirstName:      req.FirstName,
        LastName:       req.LastName,
        BillingAddress: billingAddressPtr,
        Phone:          req.Phone,
        Metadata:       req.Metadata,
    }
}

// ToCreatePaymentMethodInput converts API request to application input
func ToCreatePaymentMethodInput(req request.CreatePaymentMethodRequest, customerId string) dto.CreatePaymentMethodInput {
    // Create a pointer to the billing address
    var billingAddressPtr *entities.Address
    if (req.BillingAddress != entities.Address{}) {
        billingAddressPtr = &req.BillingAddress
    }

    return dto.CreatePaymentMethodInput{
        CustomerId:     customerId,
        Psp:            req.Psp,
        Name:           req.Name,
        Type:           req.Type,
        Details:        req.Details,
        Token:          req.Token,
        IsDefault:      req.IsDefault,
        BillingAddress: billingAddressPtr,
        Metadata:       req.Metadata,
    }
}

// ToUpdatePaymentMethodInput converts API request to application input
func ToUpdatePaymentMethodInput(req request.UpdatePaymentMethodRequest) dto.UpdatePaymentMethodInput {
    // Create pointers for optional fields
    var namePtr *string
    var isDefaultPtr *bool
    var billingAddressPtr *entities.Address

    if req.Name != "" {
        namePtr = &req.Name
    }

    isDefaultPtr = &req.IsDefault

    if (req.BillingAddress != entities.Address{}) {
        billingAddressPtr = &req.BillingAddress
    }

    return dto.UpdatePaymentMethodInput{
        Name:           namePtr,
        IsDefault:      isDefaultPtr,
        BillingAddress: billingAddressPtr,
        Metadata:       req.Metadata,
    }
}

// ToCustomerResponse converts domain entity to API response
func ToCustomerResponse(customer entities.Customer) response.Customer {
    return response.NewCustomerFromEntity(customer)
}

// ToCustomerListResponse converts paginated result to API response
func ToCustomerListResponse(result dto.PaginatedResult[entities.Customer]) response.ListResponse {
    customerResponses := make([]response.Customer, len(result.Items))
    for i, customer := range result.Items {
        customerResponses[i] = ToCustomerResponse(customer)
    }

    return response.ListResponse{
        Data: customerResponses,
        Meta: response.Meta{
            Total: result.TotalCount,
            Page:  result.Page,
            Limit: result.PageSize,
        },
    }
}