package mappers

import (
	"payloop/internal/api/dto/response"
	"payloop/internal/domain/entities"
)

// ToApiKeyResponse converts an ApiKey entity to an ApiKeyResponse DTO
func ToApiKeyResponse(apiKey entities.ApiKey) response.ApiKeyResponse {
	return response.ApiKeyResponse{
		Id:        apiKey.Id,
		Key:       apiKey.Key,
		CreatedAt: apiKey.CreatedAt,
		UpdatedAt: apiKey.UpdatedAt,
	}
}

// ToApiKeyListResponse converts a slice of ApiKey entities to an ApiKeyListResponse DTO
func ToApiKeyListResponse(apiKeys []entities.ApiKey) response.ApiKeyListResponse {
	apiKeyResponses := make([]response.ApiKeyResponse, len(apiKeys))
	for i, apiKey := range apiKeys {
		apiKeyResponses[i] = ToApiKeyResponse(apiKey)
	}
	return response.ApiKeyListResponse{
		ApiKeys: apiKeyResponses,
	}
}