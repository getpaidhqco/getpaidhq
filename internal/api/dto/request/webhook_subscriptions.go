package request

type CreateWebhookSubscriptionRequest struct {
	Url    string   `json:"url" binding:"required"`
	Events []string `json:"events" binding:"required"`
	Secret string   `json:"secret"`
}

type UpdateWebhookSubscriptionRequest struct {
	Url    string   `json:"url" binding:"required"`
	Events []string `json:"events" binding:"required"`
	Secret string   `json:"secret"`
}
