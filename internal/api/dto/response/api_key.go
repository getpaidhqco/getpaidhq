package response

import "time"

// ApiKeyResponse represents an API key response
type ApiKeyResponse struct {
	Id        string    `json:"id"`
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ApiKeyListResponse represents a list of API keys
type ApiKeyListResponse struct {
	ApiKeys []ApiKeyResponse `json:"api_keys"`
}