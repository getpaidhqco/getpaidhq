package mappers

import (
    "payloop/internal/api/dto/request"
    "payloop/internal/application/dto"
)

// ToPagination converts API pagination to application pagination
func ToPagination(req request.Pagination) dto.Pagination {
    return dto.Pagination{
        Page:          req.Page,
        Limit:         req.Limit,
        Offset:        req.Offset,
        SortDirection: req.SortDirection,
        SortBy:        req.SortBy,
        Search:        req.Search,
    }
}

// ToApiPaginatedResult converts application paginated result to API paginated result
func ToApiPaginatedResult[T any, R any](result dto.PaginatedResult[T], itemConverter func(T) R) interface{} {
    items := make([]R, len(result.Items))
    for i, item := range result.Items {
        items[i] = itemConverter(item)
    }

    return struct {
        Items      []R `json:"items"`
        TotalCount int `json:"total_count"`
        Page       int `json:"page"`
        PageSize   int `json:"page_size"`
    }{
        Items:      items,
        TotalCount: result.TotalCount,
        Page:       result.Page,
        PageSize:   result.PageSize,
    }
}
