package entities

type Product struct {
	OrgId       string             `json:"org_id"`
	Id          string             `json:"id"`
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	Metadata    *map[string]string `json:"metadata"`
}
