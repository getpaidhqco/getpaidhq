package domain

// SubscriptionStateChangedInput carries subscription state change notifications
// for dunning orchestration. Currently unused but kept for future dunning
// state-machine integration.
type SubscriptionStateChangedInput struct {
	OrgId          string
	CampaignId     string
	SubscriptionId string
	OldStatus      SubscriptionStatus
	NewStatus      SubscriptionStatus
}

// DunningAttemptContext is passed to UpdateCampaignWithAttemptResult so the
// engine adapter doesn't need to re-derive escalation state.
type DunningAttemptContext struct {
	AttemptNumber            int
	WasSubscriptionSuspended bool
}
