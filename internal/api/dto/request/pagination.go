package request

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

const (
	PageDefault  = "0"
	LimitDefault = "10"
	PageTag      = "page"
	LimitTag     = "limit"
)

// swagger:parameters listSubscriptions
type Pagination struct {
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
	SortDirection string `json:"sort_order"`
	SortBy        string `json:"sort_by"`
	Search        string `json:"search"`
}

func GetPagination(c *gin.Context) Pagination {

	page, err := strconv.Atoi(c.DefaultQuery(PageTag, PageDefault))
	if err != nil || page < 1 {
		page = 0
	}

	limit, err := strconv.Atoi(c.DefaultQuery(LimitTag, LimitDefault))
	if err != nil {
		limit = 10
	}
	sortOrder := c.DefaultQuery("sort_order", "desc")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	search := c.Query("search")

	return Pagination{
		Page:          page,
		Limit:         limit,
		Offset:        page * limit,
		SortBy:        sortBy,
		SortDirection: sortOrder,
		Search:        search,
	}
}
