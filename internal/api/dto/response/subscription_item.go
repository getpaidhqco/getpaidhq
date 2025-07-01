package response

import (
	"payloop/internal/domain/entities"
	"time"
)

// SubscriptionItemResponse represents a response containing a subscription item
type SubscriptionItemResponse struct {
	OrgId          string                      `json:"org_id"`
	Id             string                      `json:"id"`
	SubscriptionId string                      `json:"subscription_id"`
	
	// Product/Price reference
	PriceId        string                      `json:"price_id"`
	ProductId      string                      `json:"product_id,omitempty"`
	VariantId      string                      `json:"variant_id,omitempty"`
	
	// Item details
	Name           string                      `json:"name"`
	Description    string                      `json:"description,omitempty"`
	Status         entities.SubscriptionItemStatus `json:"status"`
	
	// Quantity for fixed items
	Quantity       int                         `json:"quantity"`
	
	// Billing
	Amount         int64                       `json:"amount,omitempty"`
	Currency       string                      `json:"currency"`
	
	// Usage configuration
	HasUsage       bool                        `json:"has_usage"`
	UsageType      string                      `json:"usage_type,omitempty"`
	AggregationType string                     `json:"aggregation_type,omitempty"`
	
	// Metadata
	Metadata       map[string]string           `json:"metadata,omitempty"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
}

// SubscriptionItemListResponse represents a response containing a list of subscription items
type SubscriptionItemListResponse struct {
	Items      []SubscriptionItemResponse `json:"items"`
	TotalCount int                        `json:"total_count"`
	Page       int                        `json:"page"`
	PageSize   int                        `json:"page_size"`
}

// FromSubscriptionItem converts a subscription item entity to a response
func FromSubscriptionItem(item entities.SubscriptionItem) SubscriptionItemResponse {
	return SubscriptionItemResponse{
		OrgId:          item.OrgId,
		Id:             item.Id,
		SubscriptionId: item.SubscriptionId,
		PriceId:        item.PriceId,
		ProductId:      item.ProductId,
		VariantId:      item.VariantId,
		Name:           item.Name,
		Description:    item.Description,
		Status:         item.Status,
		Quantity:       item.Quantity,
		Amount:         item.Amount,
		Currency:       item.Currency,
		HasUsage:       item.HasUsage,
		UsageType:      item.UsageType,
		AggregationType: item.AggregationType,
		Metadata:       item.Metadata,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

// FromSubscriptionItems converts a slice of subscription item entities to a response
func FromSubscriptionItems(items []entities.SubscriptionItem, totalCount, page, pageSize int) SubscriptionItemListResponse {
	var response SubscriptionItemListResponse
	response.Items = make([]SubscriptionItemResponse, len(items))
	for i, item := range items {
		response.Items[i] = FromSubscriptionItem(item)
	}
	response.TotalCount = totalCount
	response.Page = page
	response.PageSize = pageSize
	return response
}