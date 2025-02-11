package webhooks

type CreateWebhookSubscriptionInput struct {
	OrgId  string   `json:"org_id"`
	Url    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}
