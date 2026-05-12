package hatchet

import (
	"context"
	"encoding/json"
	"getpaidhq/internal/adapter/hatchet/steps"
	hatchetwf "getpaidhq/internal/adapter/hatchet/workflows"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// Hatchet implements port.Engine using Hatchet as the workflow runtime.
//
// Workflow factories are wired with the narrow services they need directly;
// there is no intermediate "steps" bundle for the order/subscription path.
// Outgoing-webhook and dunning workflows still use their own step bundles.
type Hatchet struct {
	logger        port.Logger
	client        *hatchet.Client
	worker        *hatchet.Worker
	errorReporter lib.ErrorReporter
	pubsub        port.PubSub
}

func NewHatchetEngine(
	logger port.Logger,
	env lib.Env,
	orderService port.OrderWorkflowService,
	subscriptionService port.SubscriptionService,
	paymentService port.PaymentService,
	subscriptionRepo port.SubscriptionRepository,
	errorReporter lib.ErrorReporter,
	webhookSteps *steps.OutgoingWebhookSteps,
	dunningSteps *steps.DunningSteps,
	pubsub port.PubSub,
) *Hatchet {
	logger.Infof("Initializing Hatchet engine [host_port=%s][namespace=%s]", env.HatchetHostPort, env.HatchetNamespace)

	// The Hatchet client auto-reads HATCHET_CLIENT_TOKEN, HATCHET_CLIENT_HOST_PORT,
	// HATCHET_CLIENT_NAMESPACE, HATCHET_CLIENT_TLS_STRATEGY from the environment
	// — the lib.Env values above are loaded from the same vars and are kept here
	// for visibility / future programmatic overrides.
	c, err := hatchet.NewClient()
	if err != nil {
		logger.Error("Unable to create Hatchet client", "err", err.Error())
		panic(err)
	}

	// Build the engine first so workflow definitions that need to call back
	// through the port (e.g. payment-success spawning the subscription runner)
	// can be wired with the engine reference.
	h := &Hatchet{
		logger:        logger,
		client:        c,
		errorReporter: errorReporter,
		pubsub:        pubsub,
	}

	paymentSuccessWF := hatchetwf.NewPaymentSuccessWorkflow(c, orderService, subscriptionRepo, h)
	paymentRefundedWF := hatchetwf.NewPaymentRefundedWorkflow(c, paymentService)
	outgoingWebhookWF := hatchetwf.NewOutgoingWebhookWorkflow(c, webhookSteps)
	billingCycleWF := hatchetwf.NewBillingCycleWorkflow(c, subscriptionService)
	reminderWF := hatchetwf.NewSubscriptionChargeReminderWorkflow(c, subscriptionService)
	subscriptionRunnerWF := hatchetwf.NewSubscriptionRunnerWorkflow(c, subscriptionService)
	dunningAttemptWF := hatchetwf.NewDunningAttemptWorkflow(c, dunningSteps)
	dunningRunnerWF := hatchetwf.NewDunningRunnerWorkflow(c, dunningSteps)

	w, err := c.NewWorker("getpaidhq-events",
		hatchet.WithWorkflows(
			paymentSuccessWF,
			paymentRefundedWF,
			outgoingWebhookWF,
			billingCycleWF,
			reminderWF,
			subscriptionRunnerWF,
			dunningAttemptWF,
			dunningRunnerWF,
		),
		hatchet.WithSlots(50),
		hatchet.WithDurableSlots(500),
	)
	if err != nil {
		logger.Error("Unable to create Hatchet worker", "err", err.Error())
		panic(err)
	}
	h.worker = w

	go func() {
		if err := w.StartBlocking(context.Background()); err != nil {
			logger.Error("Hatchet worker exited", "err", err.Error())
		}
	}()

	logger.Infof("Hatchet engine initialized with worker")

	_, err = pubsub.Subscribe("subscription.*", func(topic string, data []byte) {
		if err := h.HandleSubscriptionEvent(topic, data); err != nil {
			logger.Error("Failed to handle subscription event", "error", err.Error())
		}
	})
	if err != nil {
		logger.Error("Failed to subscribe to subscription.* topic", "error", err.Error())
		panic(err)
	}

	return h
}

func (h Hatchet) StartWorkflow(ctx context.Context, id port.WorkflowType, payload any) (port.WorkflowResult, error) {
	switch id {
	case port.WorkflowPaymentSuccess:
		pc, ok := payload.(domain.PaymentWebhookContext)
		if !ok {
			parsed, err := domain.ParsePaymentWebhookContext(payload)
			if err != nil {
				return port.WorkflowResult{}, err
			}
			pc = parsed
		}
		_, err := h.client.RunNoWait(ctx, "payment-success", hatchetwf.PaymentSuccessInput{PaymentContext: pc})
		if err != nil {
			h.logger.Error("Unable to run payment-success workflow", "err", err.Error())
			return port.WorkflowResult{}, err
		}
		return port.WorkflowResult{Success: true, Message: "payment-success queued"}, nil

	case port.WorkflowPaymentRefunded:
		pc, ok := payload.(domain.PaymentWebhookContext)
		if !ok {
			parsed, err := domain.ParsePaymentWebhookContext(payload)
			if err != nil {
				return port.WorkflowResult{}, err
			}
			pc = parsed
		}
		_, err := h.client.RunNoWait(ctx, "payment-refunded", hatchetwf.PaymentRefundedInput{PaymentContext: pc})
		if err != nil {
			h.logger.Error("Unable to run payment-refunded workflow", "err", err.Error())
			return port.WorkflowResult{}, err
		}
		return port.WorkflowResult{Success: true, Message: "payment-refunded queued"}, nil

	case port.WorkflowOutgoingWebhook:
		wh, ok := payload.(port.OutgoingWebhookPayload)
		if !ok {
			return port.WorkflowResult{}, &portError{Msg: "outgoing-webhook expects port.OutgoingWebhookPayload"}
		}
		_, err := h.client.RunNoWait(ctx, "outgoing-webhook", wh)
		if err != nil {
			h.logger.Error("Unable to run outgoing-webhook workflow", "err", err.Error())
			return port.WorkflowResult{}, err
		}
		return port.WorkflowResult{Success: true, Message: "outgoing-webhook queued"}, nil

	default:
		h.logger.Warnf("Unsupported workflow type: %s", id)
		return port.WorkflowResult{}, nil
	}
}

func (h Hatchet) StartSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	ref, err := h.client.RunNoWait(ctx, "subscription-runner", sub,
		hatchet.WithRunKey(hatchetwf.SubscriptionRunKey(sub.OrgId, sub.Id)),
	)
	if err != nil {
		h.logger.Error("Unable to run subscription-runner", "err", err.Error())
		return err
	}
	h.logger.Info("Started subscription-runner", "RunID", ref.RunId, "OrgId", sub.OrgId, "SubscriptionId", sub.Id)
	return nil
}

func (h Hatchet) UpdateSubscriptionWorkflow(ctx context.Context, updateName string, sub domain.Subscription) error {
	key := hatchetwf.UpdateEventKey(updateName, sub.OrgId, sub.Id)
	h.logger.Debugf("Pushing update event [%s]", key)
	if err := h.client.Events().Push(ctx, key, sub); err != nil {
		h.logger.Error("Failed to push update event", "error", err.Error(), "key", key)
		h.errorReporter.ReportError(ctx, err, map[string]any{
			"org_id":          sub.OrgId,
			"subscription_id": sub.Id,
			"update_name":     updateName,
		})
		return err
	}
	return nil
}

func (h Hatchet) CancelSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	key := hatchetwf.CancelEventKey(sub.OrgId, sub.Id)
	h.logger.Debugf("Pushing cancel event [%s]", key)
	if err := h.client.Events().Push(ctx, key, sub); err != nil {
		h.logger.Error("Failed to push cancel event", "error", err.Error(), "key", key)
		h.errorReporter.ReportError(ctx, err, map[string]any{
			"org_id":          sub.OrgId,
			"subscription_id": sub.Id,
		})
		return err
	}
	return nil
}

func (h Hatchet) SignalSubscriptionWorkflow(ctx context.Context, signal string, sub domain.Subscription, payload any) error {
	var key string
	switch signal {
	case "webhook-signal":
		key = hatchetwf.WebhookEventKey(sub.OrgId, sub.Id)
	default:
		key = hatchetwf.UpdateEventKey(signal, sub.OrgId, sub.Id)
	}
	h.logger.Debugf("Pushing signal event [%s]", key)
	if err := h.client.Events().Push(ctx, key, payload); err != nil {
		h.logger.Error("Failed to push signal event", "error", err.Error(), "key", key)
		return err
	}
	return nil
}

// HandleSubscriptionEvent fans incoming "subscription.*" pubsub topics into
// Hatchet update events so the per-subscription durable runner picks them up.
// Mirrors the Temporal adapter's HandleSubscriptionEvent.
func (h Hatchet) HandleSubscriptionEvent(topic string, data []byte) error {
	h.logger.Infof("Received topic [%s]", topic)

	var eventData port.PubSubPayload
	if err := json.Unmarshal(data, &eventData); err != nil {
		h.logger.Error("Failed to unmarshal event data", "error", err.Error())
		return err
	}

	dataBytes, err := json.Marshal(eventData.Data)
	if err != nil {
		h.logger.Error("Failed to marshal subscription data", "error", err.Error())
		return err
	}
	var sub domain.Subscription
	if err := json.Unmarshal(dataBytes, &sub); err != nil {
		h.logger.Error("Failed to unmarshal event data to Subscription", "error", err.Error())
		return err
	}

	switch topic {
	case "subscription.paused":
		return h.UpdateSubscriptionWorkflow(context.Background(), topic, sub)
	default:
		h.logger.Infof("No handler for topic %s", topic)
	}
	return nil
}

type portError struct{ Msg string }

func (e *portError) Error() string { return e.Msg }

// ---- port.DunningEngine implementation ----

// StartDunningWorkflow spawns the dunning-runner durable task for the
// caller-created campaign. Returns (workflowName, runId) which the
// orchestrator persists on the campaign for later signaling/cancellation.
//
// The orchestrator passes input.Metadata["campaign_id"] to identify which
// campaign this run is for; if absent we fall back to subscriptionId-based
// idempotency.
func (h Hatchet) StartDunningWorkflow(ctx context.Context, input domain.StartDunningWorkflowInput) (string, string, error) {
	campaignId := ""
	if input.Metadata != nil {
		campaignId = input.Metadata["campaign_id"]
	}
	if campaignId == "" {
		// Fallback: subscription-scoped key (the campaign id will be filled in
		// by the runner via its first activity if we ever support that flow).
		campaignId = input.SubscriptionId
	}
	runnerInput := hatchetwf.DunningRunnerInput{
		OrgId:                input.OrgId,
		CampaignId:           campaignId,
		SubscriptionId:       input.SubscriptionId,
		CustomerId:           input.CustomerId,
		FailedAmount:         input.FailedAmount,
		Currency:             input.Currency,
		InitialFailureReason: input.InitialFailureReason,
		PaymentResult:        input.PaymentResult,
		Metadata:             input.Metadata,
	}
	ref, err := h.client.RunNoWait(ctx, "dunning-runner", runnerInput,
		hatchet.WithRunKey(hatchetwf.DunningRunKey(input.OrgId, campaignId)),
	)
	if err != nil {
		h.logger.Error("Unable to start dunning-runner", "err", err.Error())
		return "", "", err
	}
	h.logger.Info("Started dunning-runner", "RunID", ref.RunId, "OrgId", input.OrgId, "CampaignId", campaignId)
	return "dunning-runner", ref.RunId, nil
}

func (h Hatchet) SignalDunningWorkflow(ctx context.Context, signal string, campaign domain.DunningCampaign, payload any) error {
	var key string
	switch signal {
	case "payment_method.updated":
		key = hatchetwf.DunningPaymentMethodUpdatedKey(campaign.OrgId, campaign.Id)
	default:
		key = hatchetwf.DunningSignalKey(signal, campaign.OrgId, campaign.Id)
	}
	h.logger.Debugf("Pushing dunning signal [%s]", key)
	if err := h.client.Events().Push(ctx, key, payload); err != nil {
		h.logger.Error("Failed to push dunning signal", "error", err.Error(), "key", key)
		return err
	}
	return nil
}

func (h Hatchet) CancelDunningWorkflow(ctx context.Context, campaign domain.DunningCampaign) error {
	key := hatchetwf.DunningSignalKey("dunning.cancel", campaign.OrgId, campaign.Id)
	h.logger.Debugf("Pushing dunning cancel [%s]", key)
	if err := h.client.Events().Push(ctx, key, campaign); err != nil {
		h.logger.Error("Failed to push dunning cancel", "error", err.Error(), "key", key)
		return err
	}
	return nil
}
