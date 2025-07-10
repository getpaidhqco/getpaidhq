package mappers

import (
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

// ToCreateMeterInput converts API request to application input
func ToCreateMeterInput(req request.CreateMeterRequest) dto.CreateMeterInput {
	return dto.CreateMeterInput{
		Name:            req.Name,
		Description:     req.Description,
		EventName:       req.EventName,
		EventFilter:     req.EventFilter,
		AggregationType: entities.AggregationType(req.AggregationType),
		ValueProperty:   req.ValueProperty,
		UnitType:        entities.UnitType(req.UnitType),
		DisplayName:     req.DisplayName,
		WindowSize:      req.WindowSize,
		ResetInterval:   req.ResetInterval,
		Metadata:        req.Metadata,
	}
}

// ToUpdateMeterInput converts API request to application input
func ToUpdateMeterInput(req request.UpdateMeterRequest) dto.UpdateMeterInput {
	return dto.UpdateMeterInput{
		Name:            req.Name,
		Description:     req.Description,
		EventName:       req.EventName,
		EventFilter:     req.EventFilter,
		AggregationType: entities.AggregationType(req.AggregationType),
		ValueProperty:   req.ValueProperty,
		UnitType:        entities.UnitType(req.UnitType),
		DisplayName:     req.DisplayName,
		WindowSize:      req.WindowSize,
		ResetInterval:   req.ResetInterval,
		Metadata:        req.Metadata,
	}
}

// ToMeterResponse converts domain entity to API response
func ToMeterResponse(meter entities.Meter) response.Meter {
	return response.NewMeterFromEntity(meter)
}

// ToMeterListResponse converts paginated result to API response
func ToMeterListResponse(result dto.PaginatedResult[entities.Meter]) response.ListResponse {
	meters := make([]response.Meter, len(result.Items))
	for i, meter := range result.Items {
		meters[i] = ToMeterResponse(meter)
	}

	return response.ListResponse{
		Data: meters,
		Meta: response.Meta{
			Total: result.TotalCount,
			Page:  result.Page,
			Limit: result.PageSize,
		},
	}
}