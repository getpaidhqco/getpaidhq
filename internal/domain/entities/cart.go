package entities

type Cart struct {
	OrgId    string      `json:"org_id"`
	Id       string      `json:"id"`
	Data     interface{} `json:"data"`
	Status   string      `json:"status"`
	Total    int64       `json:"total"`
	Metadata interface{} `json:"metadata"`
}

type CartStatus string

const (
	CartStatusPending   CartStatus = "pending"
	CartStatusCompleted CartStatus = "completed"
	CartStatusExpired   CartStatus = "expired"
)
