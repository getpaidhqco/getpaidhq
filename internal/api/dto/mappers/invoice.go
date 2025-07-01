package mappers

import (
    "payloop/internal/api/dto/request"
    "payloop/internal/domain/entities"
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