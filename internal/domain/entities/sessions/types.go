package sessions

type CreateSessionInput struct {
	OrgId    string            `json:"org_id"`
	Id       string            `json:"id"`
	CartId   string            `json:"cart_id"`
	Metadata map[string]string `json:"metadata"`
}

type CreateSessionRequest struct {
	OrgId    string            `json:"org_id"`
	Currency string            `json:"currency"`
	Country  string            `json:"country"`
	Metadata map[string]string `json:"metadata"`
}

type CreateSessionResponse struct {
	Id     string `json:"id"`
	CartId string `json:"cart_id"`
}
