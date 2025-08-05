package mappers

import (
    "encoding/json"
    "payloop/internal/api/dto/request"
    "payloop/internal/api/dto/response"
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
    "payloop/internal/domain/entities/orders"
)

// ToCreateInvoiceInput converts API request to domain input
func ToCreateInvoiceInput(req request.CreateInvoiceRequest) entities.CreateInvoiceInput {
    input := entities.CreateInvoiceInput{
        CustomerId:     req.CustomerId,
        OrderId:        req.OrderId,
        SubscriptionId: req.SubscriptionId,
        Type:           req.Type,
        InvoiceType:    req.InvoiceType,
        Currency:       req.Currency,
        DueAt:          req.DueAt,
        Notes:          req.Notes,
        CustomerNotes:  req.CustomerNotes,
        Metadata:       req.Metadata,
    }

    // Convert line items
    if len(req.LineItems) > 0 {
        input.LineItems = make([]entities.CreateInvoiceLineItemInput, len(req.LineItems))
        for i, item := range req.LineItems {
            input.LineItems[i] = ToCreateInvoiceLineItemInput(item)
        }
    }

    return input
}

// ToUpdateInvoiceInput converts API request to domain input
func ToUpdateInvoiceInput(req request.UpdateInvoiceRequest) entities.UpdateInvoiceInput {
    return entities.UpdateInvoiceInput{
        Notes:         req.Notes,
        CustomerNotes: req.CustomerNotes,
        DueAt:         req.DueAt,
        Metadata:      req.Metadata,
    }
}

// ToCreateInvoiceLineItemInput converts API request to domain input
func ToCreateInvoiceLineItemInput(req request.CreateInvoiceLineItemRequest) entities.CreateInvoiceLineItemInput {
    return entities.CreateInvoiceLineItemInput{
        ProductId:     req.ProductId,
        VariantId:     req.VariantId,
        PriceId:       req.PriceId,
        Description:   req.Description,
        Category:      req.Category,
        Quantity:      req.Quantity,
        UnitPrice:     req.UnitPrice,
        DiscountType:  req.DiscountType,
        DiscountValue: req.DiscountValue,
        TaxCode:       req.TaxCode,
        TaxRate:       req.TaxRate,
        TaxExempt:     req.TaxExempt,
        Metadata:      req.Metadata,
    }
}

// ToUpdateInvoiceLineItemInput converts API request to domain input
func ToUpdateInvoiceLineItemInput(req request.UpdateInvoiceLineItemRequest) entities.UpdateInvoiceLineItemInput {
    return entities.UpdateInvoiceLineItemInput{
        Description:   req.Description,
        Category:      req.Category,
        Quantity:      req.Quantity,
        UnitPrice:     req.UnitPrice,
        DiscountType:  req.DiscountType,
        DiscountValue: req.DiscountValue,
        TaxCode:       req.TaxCode,
        TaxRate:       req.TaxRate,
        TaxExempt:     req.TaxExempt,
        Metadata:      req.Metadata,
    }
}

// ToInvoiceActionInput converts API request to domain input
func ToInvoiceActionInput(req request.InvoiceActionRequest) entities.InvoiceActionInput {
    return entities.InvoiceActionInput{
        Action: req.Action,
        Reason: req.Reason,
    }
}

// ToGenerateInvoicePDFInput converts API request to domain input
func ToGenerateInvoicePDFInput(req request.GenerateInvoicePDFRequest) entities.GenerateInvoicePDFInput {
    return entities.GenerateInvoicePDFInput{
        TemplateName: req.TemplateName,
        OutputPath:   req.OutputPath,
    }
}

// ToInitiatePaymentInput converts API request to application DTO
func ToInitiatePaymentInput(req request.InitiateInvoicePaymentRequest) dto.InitiatePaymentInput {
    return dto.InitiatePaymentInput{
        PaymentProcessor: req.PaymentProcessor,
        BillingAddress:   req.BillingAddress,
        SuccessUrl:       req.SuccessUrl,
        CancelUrl:        req.CancelUrl,
        Metadata:         req.Metadata,
    }
}

// ToInitiatePaymentResponse converts order and PSP response to API response
func ToInitiatePaymentResponse(order entities.Order, pspResponse orders.CreateOrderResponse, paymentProcessor string) response.InitiatePaymentResponse {
    resp := response.InitiatePaymentResponse{
        OrderId:          order.Id,
        PaymentProcessor: paymentProcessor,
        Amount:           order.Total,
        Currency:         order.Currency,
        Status:           "pending",
        Reference:        order.Reference,
    }

    // Extract PSP-specific data from the response (stored as JSON in the PSP response)
    if pspResponse.Psp.PspResponse != nil {
        // Try to convert to JSON first to handle different PSP response structures
        if jsonData, err := json.Marshal(pspResponse.Psp.PspResponse); err == nil {
            var pspData map[string]interface{}
            if json.Unmarshal(jsonData, &pspData) == nil {
                // For Paystack - extract authorization_url
                if authUrl, exists := pspData["authorization_url"]; exists {
                    if url, ok := authUrl.(string); ok {
                        resp.RedirectUrl = url
                    }
                }

                // For other PSPs - extract client_secret, session_id etc.
                if clientSecret, exists := pspData["client_secret"]; exists {
                    if secret, ok := clientSecret.(string); ok {
                        resp.ClientSecret = secret
                    }
                }

                if sessionId, exists := pspData["session_id"]; exists {
                    if id, ok := sessionId.(string); ok {
                        resp.SessionId = id
                    }
                }
            }
        }
    }

    return resp
}