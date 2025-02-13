package sessions

type CreateSessionInput struct {
	OrgId    string            `json:"org_id"`
	Currency string            `json:"currency" binding:"required"`
	Country  string            `json:"country" binding:"required"`
	Metadata map[string]string `json:"metadata"`
}

type CreateSessionRequest struct {
	Currency string            `json:"currency" binding:"required"`
	Country  string            `json:"country" binding:"required"`
	Metadata map[string]string `json:"metadata"`
}

type CreateSessionResponse struct {
	Id     string `json:"id"`
	CartId string `json:"cart_id"`
}
