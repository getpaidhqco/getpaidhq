package port

// CreateWebhookSubscriptionInput is the command input for WebhookSubscriptionService.Create.
type CreateWebhookSubscriptionInput struct {
	OrgId  string
	Url    string
	Events []string
	Secret string
}
