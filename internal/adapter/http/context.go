package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/go-fuego/fuego/param"

	"getpaidhq/internal/adapter/http/middleware"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// PaginationParams declares the standard pagination/sorting query parameters
// in the OpenAPI spec for a list route. Pair it with GetPagination, which
// reads the same four keys; declaring them here keeps Fuego from logging
// "query parameter not expected in OpenAPI spec" warnings.
func PaginationParams() []fuego.RouteOption {
	return []fuego.RouteOption{
		option.QueryInt("page", "Page number (0-based)", param.Default(0)),
		option.QueryInt("limit", "Items per page", param.Default(10)),
		option.Query("sort_by", "Field to sort by", param.Default("created_at")),
		option.Query("sort_order", "Sort direction (asc or desc)", param.Default("desc")),
	}
}

// AuthUserFrom reads the authenticated user from the Fuego context.
// AuthnWrapperMiddleware stores it on the request context; this helper
// keeps the call sites in handlers a single line.
func AuthUserFrom[B, P any](c fuego.Context[B, P]) port.AuthUser {
	u, _ := middleware.AuthUserFrom(c.Context())
	return u
}

// GetPagination reads the standard pagination/sorting query parameters
// from the request and returns the domain Pagination value. The same
// four query keys (page, limit, sort_by, sort_order) used to be parsed
// in the gin handler.
func GetPagination[B, P any](c fuego.Context[B, P]) domain.Pagination {
	page := c.QueryParamInt("page")
	if page < 0 {
		page = 0
	}
	limit, err := c.QueryParamIntErr("limit")
	if err != nil || limit <= 0 {
		limit = 10
	}
	sortOrder := c.QueryParam("sort_order")
	if sortOrder == "" {
		sortOrder = "desc"
	}
	sortBy := c.QueryParam("sort_by")
	if sortBy == "" {
		sortBy = "created_at"
	}

	return domain.Pagination{
		Page:          page,
		Limit:         limit,
		Offset:        page * limit,
		SortBy:        sortBy,
		SortDirection: sortOrder,
	}
}
