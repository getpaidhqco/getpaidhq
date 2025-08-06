package mappers

import (
	"encoding/json"
	"fmt"
	"payloop/internal/api/dto/response"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
)

// ToPublicInvoiceResponse converts domain Invoice to public response
func ToPublicInvoiceResponse(invoice entities.Invoice) *response.PublicInvoiceResponse {
	var lineItems []response.PublicInvoiceLineItem
	for _, item := range invoice.LineItems {
		lineItems = append(lineItems, response.PublicInvoiceLineItem{
			Description: item.Description,
			Quantity:    fmt.Sprintf("%.3f", item.Quantity),
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.LineTotal,
		})
	}

	// Note: Invoice entity doesn't have direct Customer field
	// Customer data would need to be fetched separately if needed
	var customer *response.PublicCustomerResponse = nil

	var dueAt *string
	if !invoice.DueAt.IsZero() {
		dueAtStr := invoice.DueAt.Format("2006-01-02")
		dueAt = &dueAtStr
	}

	return &response.PublicInvoiceResponse{
		Id:        invoice.Id,
		DocNumber: invoice.DocNumber,
		Total:     invoice.Total,
		AmountDue: fmt.Sprintf("%d", invoice.AmountDue),
		Currency:  invoice.Currency,
		DueAt:     dueAt,
		LineItems: lineItems,
		Customer:  customer,
	}
}

// ToPublicOrderResponse converts order creation response to public response  
func ToPublicOrderResponse(order entities.Order, orderResponse orders.CreateOrderResponse, paymentProcessor string) response.PublicOrderResponse {
	resp := response.PublicOrderResponse{
		OrderId:          order.Id,
		PaymentProcessor: paymentProcessor,
		Reference:        order.Reference,
		Amount:           int(order.Total),
		Currency:         order.Currency,
		Status:           string(order.Status),
	}

	// Extract PSP-specific data from the response
	if orderResponse.Psp.PspResponse != nil {
		// Try to convert to JSON first to handle different PSP response structures
		if jsonData, err := json.Marshal(orderResponse.Psp.PspResponse); err == nil {
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