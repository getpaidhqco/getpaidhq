package mappers

import (
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"time"
)

// ToCreateSubscriptionInput converts API request to application DTO
func ToCreateSubscriptionInput(req request.CreateSubscriptionRequest) dto.CreateSubscriptionInput {
	items := make([]dto.CreateSubscriptionItemInput, len(req.Items))
	for i, item := range req.Items {
		items[i] = dto.CreateSubscriptionItemInput{
			PriceId:     item.PriceId,
			Name:        item.Name,
			Description: item.Description,
			Quantity:    item.Quantity,
			Metadata:    item.Metadata,
		}
	}

	return dto.CreateSubscriptionInput{
		CustomerId:      req.CustomerId,
		PaymentMethodId: req.PaymentMethodId,
		Currency:        req.Currency,
		Items:           items,
		Metadata:        req.Metadata,
	}
}

// ToUpdateSubscriptionInput converts API request to domain input
func ToUpdateSubscriptionInput(req request.UpdateSubscriptionRequest) entities.UpdateSubscriptionInput {
	return entities.UpdateSubscriptionInput{
		PaymentMethodId: req.PaymentMethodId,
		Metadata:        req.Metadata,
	}
}

// ToPauseSubscriptionInput converts API request to domain input
func ToPauseSubscriptionInput(req request.PauseSubscriptionRequest) (entities.PauseSubscriptionInput, error) {
	input := entities.PauseSubscriptionInput{
		PauseMode: req.PauseMode,
	}

	if req.ResumeAt != "" {
		resumeAt, err := time.Parse(time.RFC3339, req.ResumeAt)
		if err != nil {
			return input, err
		}
		input.ResumeAt = resumeAt
	}

	return input, nil
}

// ToResumeSubscriptionInput converts API request to domain input
func ToResumeSubscriptionInput(req request.ResumeSubscriptionRequest) entities.ResumeSubscriptionInput {
	return entities.ResumeSubscriptionInput{
		ProrationMode: req.ProrationMode,
	}
}

// ToCancelSubscriptionInput converts API request to domain input
func ToCancelSubscriptionInput(req request.CancelSubscriptionRequest) (entities.CancelSubscriptionInput, error) {
	input := entities.CancelSubscriptionInput{
		CancelMode:    req.CancelMode,
		ProrationMode: req.ProrationMode,
	}

	if req.CancellationDate != "" {
		cancelDate, err := time.Parse(time.RFC3339, req.CancellationDate)
		if err != nil {
			return input, err
		}
		input.CancellationDate = cancelDate
	}

	return input, nil
}

// ToChangePlanInput converts API request to domain input
func ToChangePlanInput(req request.ChangePlanRequest) (entities.ChangePlanInput, error) {
	input := entities.ChangePlanInput{
		NewPriceId:    req.NewPriceId,
		ProrationMode: req.ProrationMode,
	}

	if req.EffectiveDate != "" {
		effectiveDate, err := time.Parse(time.RFC3339, req.EffectiveDate)
		if err != nil {
			return input, err
		}
		input.EffectiveDate = effectiveDate
	}

	return input, nil
}
