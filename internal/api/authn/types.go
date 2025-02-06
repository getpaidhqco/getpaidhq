package authn

type User struct {
	OrgId string   `json:"org_id"`
	Id    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"role"`
}
