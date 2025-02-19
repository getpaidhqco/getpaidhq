package entities

type EntityKey struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"`
}

type Pagination struct {
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
	SortDirection string `json:"sort_order"`
	SortBy        string `json:"sort_by"`
}
