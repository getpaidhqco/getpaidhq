package dto

// Pagination represents pagination parameters for list operations
type Pagination struct {
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
	SortDirection string `json:"sort_order"`
	SortBy        string `json:"sort_by"`
}

// NewPagination creates a new Pagination instance with default values
func NewPagination(page, limit int, sortBy, sortDirection string) Pagination {
	if page < 1 {
		page = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortDirection == "" {
		sortDirection = "desc"
	}
	
	return Pagination{
		Page:          page,
		Limit:         limit,
		Offset:        page * limit,
		SortBy:        sortBy,
		SortDirection: sortDirection,
	}
}