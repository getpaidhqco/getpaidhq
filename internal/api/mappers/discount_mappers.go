package mappers

import (
	"encoding/json"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"time"
)

// ToCreateDiscountInput converts a CreateDiscountRequest to a CreateDiscountInput
func ToCreateDiscountInput(req request.CreateDiscountRequest) dto.CreateDiscountInput {
	return dto.CreateDiscountInput{
		Name:           req.Name,
		Type:           req.Type,
		Value:          req.Value,
		Code:           req.Code,
		StartsAt:       req.StartsAt,
		EndsAt:         req.EndsAt,
		MaxRedemptions: req.MaxRedemptions,
		Recurring:      req.Recurring,
		Cycles:         req.Cycles,
		Currency:       req.Currency,
		Active:         req.Active,
		Metadata:       req.Metadata,
	}
}

// ToUpdateDiscountInput converts an UpdateDiscountRequest to an UpdateDiscountInput
func ToUpdateDiscountInput(req request.UpdateDiscountRequest) dto.UpdateDiscountInput {
	return dto.UpdateDiscountInput{
		Name:           req.Name,
		Type:           req.Type,
		Value:          req.Value,
		Code:           req.Code,
		StartsAt:       req.StartsAt,
		EndsAt:         req.EndsAt,
		MaxRedemptions: req.MaxRedemptions,
		Recurring:      req.Recurring,
		Cycles:         req.Cycles,
		Currency:       req.Currency,
		Active:         req.Active,
		Metadata:       req.Metadata,
	}
}

// ToValidateDiscountCodeInput converts a ValidateDiscountCodeRequest to a ValidateDiscountCodeInput
func ToValidateDiscountCodeInput(req request.ValidateDiscountCodeRequest) dto.ValidateDiscountCodeInput {
	return dto.ValidateDiscountCodeInput{
		Code:       req.Code,
		CustomerId: req.CustomerId,
		Amount:     req.Amount,
		Currency:   req.Currency,
	}
}

// ToApplyDiscountInput converts an ApplyDiscountRequest to an ApplyDiscountInput
func ToApplyDiscountInput(req request.ApplyDiscountRequest) dto.ApplyDiscountInput {
	return dto.ApplyDiscountInput{
		DiscountId:   req.DiscountId,
		CustomerId:   req.CustomerId,
		ResourceType: req.ResourceType,
		ResourceId:   req.ResourceId,
		Amount:       req.Amount,
		Currency:     req.Currency,
		Metadata:     req.Metadata,
	}
}

// ToDiscountResponse converts a Discount entity to a DiscountResponse
func ToDiscountResponse(discount entities.Discount) response.DiscountResponse {
	var metadata map[string]interface{}
	if len(discount.Metadata) > 0 {
		_ = json.Unmarshal(discount.Metadata, &metadata)
	}

	var startsAt, endsAt *time.Time
	if !discount.StartsAt.IsZero() {
		startsAt = &discount.StartsAt
	}
	if !discount.EndsAt.IsZero() {
		endsAt = &discount.EndsAt
	}

	return response.DiscountResponse{
		Id:             discount.Id,
		OrgId:          discount.OrgId,
		Name:           discount.Name,
		Type:           discount.Type,
		Value:          discount.Value,
		Code:           discount.Code,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		MaxRedemptions: discount.MaxRedemptions,
		Recurring:      discount.Recurring,
		Cycles:         discount.Cycles,
		Currency:       discount.Currency,
		Active:         discount.Active,
		CreatedAt:      discount.CreatedAt,
		UpdatedAt:      discount.UpdatedAt,
		Metadata:       metadata,
	}
}

// ToDiscountListResponse converts a PaginatedResult of Discount entities to a DiscountListResponse
func ToDiscountListResponse(result dto.PaginatedResult[entities.Discount]) response.DiscountListResponse {
	items := make([]response.DiscountResponse, len(result.Items))
	for i, discount := range result.Items {
		items[i] = ToDiscountResponse(discount)
	}

	return response.DiscountListResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PageSize:   result.PageSize,
	}
}

// ToDiscountRedemptionResponse converts a DiscountRedemption entity to a DiscountRedemptionResponse
func ToDiscountRedemptionResponse(redemption entities.DiscountRedemption) response.DiscountRedemptionResponse {
	var metadata map[string]interface{}
	if len(redemption.Metadata) > 0 {
		_ = json.Unmarshal(redemption.Metadata, &metadata)
	}

	return response.DiscountRedemptionResponse{
		Id:             redemption.Id,
		OrgId:          redemption.OrgId,
		DiscountId:     redemption.DiscountId,
		CustomerId:     redemption.CustomerId,
		ResourceType:   redemption.ResourceType,
		ResourceId:     redemption.ResourceId,
		DiscountAmount: redemption.DiscountAmount,
		Currency:       redemption.Currency,
		CreatedAt:      redemption.CreatedAt,
		Metadata:       metadata,
	}
}

// ToDiscountRedemptionListResponse converts a PaginatedResult of DiscountRedemption entities to a DiscountRedemptionListResponse
func ToDiscountRedemptionListResponse(result dto.PaginatedResult[entities.DiscountRedemption]) response.DiscountRedemptionListResponse {
	items := make([]response.DiscountRedemptionResponse, len(result.Items))
	for i, redemption := range result.Items {
		items[i] = ToDiscountRedemptionResponse(redemption)
	}

	return response.DiscountRedemptionListResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PageSize:   result.PageSize,
	}
}

// ToDiscountValidationResponse converts a DiscountValidationResult to a DiscountValidationResponse
func ToDiscountValidationResponse(result dto.DiscountValidationResult) response.DiscountValidationResponse {
	return response.DiscountValidationResponse{
		Valid:          result.Valid,
		DiscountId:     result.DiscountId,
		DiscountAmount: result.DiscountAmount,
		Message:        result.Message,
	}
}

// ToDiscountRedemptionResultResponse converts a DiscountRedemptionResult to a DiscountRedemptionResultResponse
func ToDiscountRedemptionResultResponse(result dto.DiscountRedemptionResult) response.DiscountRedemptionResultResponse {
	return response.DiscountRedemptionResultResponse{
		RedemptionId:     result.RedemptionId,
		DiscountId:       result.DiscountId,
		DiscountAmount:   result.DiscountAmount,
		OriginalAmount:   result.OriginalAmount,
		DiscountedAmount: result.DiscountedAmount,
		Currency:         result.Currency,
	}
}