package entities

type Customer struct {
	OrgId string `json:"org_id"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
