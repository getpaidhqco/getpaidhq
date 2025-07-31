package dto

// PaginatedResult represents a paginated result set
type PaginatedResult[T any] struct {
    Items      []T `json:"items"`
    TotalCount int `json:"total_count"`
    Page       int `json:"page"`
    PageSize   int `json:"page_size"`
}