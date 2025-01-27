package entities

type Setting struct {
	OrgId string         `json:"org_id"`
	Id    string         `json:"id"`
	Value map[string]any `json:"value"`
}
