package request

import (
	"payloop/internal/application/dto"
)

// ToUpdateInvoiceRequestDTO converts a request.UpdateInvoiceRequest to a dto.UpdateInvoiceRequest
func (r UpdateInvoiceRequest) ToDTO() dto.UpdateInvoiceRequest {
	return dto.UpdateInvoiceRequest{
		Notes:         r.Notes,
		CustomerNotes: r.CustomerNotes,
		DueAt:         r.DueAt,
		Metadata:      r.Metadata,
	}
}

// ToPaginationDTO converts a request.Pagination to a dto.Pagination
func (p Pagination) ToDTO() dto.Pagination {
	return dto.Pagination{
		Page:          p.Page,
		Limit:         p.Limit,
		Offset:        p.Offset,
		SortDirection: p.SortDirection,
		SortBy:        p.SortBy,
	}
}

// ToInvoiceActionRequestDTO converts a request.InvoiceActionRequest to a dto.InvoiceActionRequest
func (r InvoiceActionRequest) ToDTO() dto.InvoiceActionRequest {
	return dto.InvoiceActionRequest{
		Action: r.Action,
		Reason: r.Reason,
	}
}

// ToUpdateInvoiceLineItemRequestDTO converts a request.UpdateInvoiceLineItemRequest to a dto.UpdateInvoiceLineItemRequest
func (r UpdateInvoiceLineItemRequest) ToDTO() dto.UpdateInvoiceLineItemRequest {
	return dto.UpdateInvoiceLineItemRequest{
		Description:   r.Description,
		Category:      r.Category,
		Quantity:      r.Quantity,
		UnitPrice:     r.UnitPrice,
		DiscountType:  r.DiscountType,
		DiscountValue: r.DiscountValue,
		TaxCode:       r.TaxCode,
		TaxRate:       r.TaxRate,
		TaxExempt:     r.TaxExempt,
		Metadata:      r.Metadata,
	}
}