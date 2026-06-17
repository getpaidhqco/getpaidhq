package hatchet

import (
	"context"
	"errors"
	"time"

	"getpaidhq/internal/adapter/hatchet/steps"
	hatchetwf "getpaidhq/internal/adapter/hatchet/workflows"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	hatchetclient "github.com/hatchet-dev/hatchet/pkg/client"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	hatchetotel "github.com/hatchet-dev/hatchet/sdks/go/opentelemetry"
)

// Hatchet implements port.Engine and port.DunningEngine using Hatchet as the
// workflow runtime.
//
// Workflow factories are wired with the narrow services they need directly;
// there is no intermediate "steps" bundle for the order/subscription path.
// Outgoing-webhook and dunning workflows still use their own step bundles.
//
// Pubsub fan-in to engine signals is handled by SubscriptionEventBridge in
// the service layer, not by this adapter.
type Hatchet struct {
	logger port.Logger
	client *hatchet.Client
	worker *hatchet.Worker
	cancel context.CancelFunc
	done   chan struct{}
	// instrumentor is the SDK's OTel tracer setup; non-nil only when
	// cfg.TracingEnabled. Shut down (flushing pending spans) in Close.
	instrumentor *hatchetotel.Instrumentor
}

func NewHatchetEngine(
	logger port.Logger,
	cfg Config,
	orderService port.OrderWorkflowService,
	subscriptionService port.SubscriptionService,
	paymentService port.PaymentService,
	subscriptionRepo port.SubscriptionRepository,
	orgRepo port.OrgRepository,
	reminderResolver port.ReminderConfigResolver,
	webhookSteps *steps.OutgoingWebhookSteps,
	dunningSteps *steps.DunningSteps,
) *Hatchet {
	logger.Infof("Initializing Hatchet engine [host_port=%s][namespace=%s]", cfg.HostPort, cfg.Namespace)

	// The Hatchet client auto-reads HATCHET_CLIENT_TOKEN, HATCHET_CLIENT_HOST_PORT,
	// HATCHET_CLIENT_NAMESPACE, HATCHET_CLIENT_TLS_STRATEGY from the environment
	// — the Config values above are loaded from the same vars and are kept here
	// for visibility / future programmatic overrides.
	// Bridge Hatchet's zerolog output into the app logger so the client's
	// heartbeat/connection chatter shares our slog format instead of writing
	// its own JSON to stderr. Filtered by cfg.LogLevel, not the app level.
	hatchetLog := newZerologToSlog(logger, cfg.LogLevel)
	c, err := hatchet.NewClient(hatchetclient.WithLogger(&hatchetLog))
	if err != nil {
		logger.Error("Unable to create Hatchet client", "err", err.Error())
		panic(err)
	}

	// Build the engine first so workflow definitions that need to call back
	// through the port (e.g. payment-success spawning the subscription runner)
	// can be wired with the engine reference.
	h := &Hatchet{
		logger: logger,
		client: c,
	}

	paymentSuccessWF := hatchetwf.NewPaymentSuccessWorkflow(c, orderService, subscriptionRepo, h)
	paymentRefundedWF := hatchetwf.NewPaymentRefundedWorkflow(c, paymentService)
	outgoingWebhookWF := hatchetwf.NewOutgoingWebhookWorkflow(c, webhookSteps)
	billingCycleWF := hatchetwf.NewBillingCycleWorkflow(c, subscriptionService)
	billingCycleRunnerWF := hatchetwf.NewBillingCycleRunnerWorkflow(c, subscriptionService)
	orgBillingWF := hatchetwf.NewOrgBillingWorkflow(c, subscriptionRepo, reminderResolver, logger)
	billingSweepWF := hatchetwf.NewBillingSweepWorkflow(c, orgRepo, cfg.BillingSweepInterval, logger)
	sendReminderWF := hatchetwf.NewSendRenewalReminderWorkflow(c, subscriptionService)
	dunningAttemptWF := hatchetwf.NewDunningAttemptWorkflow(c, dunningSteps)
	dunningRunnerWF := hatchetwf.NewDunningRunnerWorkflow(c, dunningSteps)
	dunningCommunicationWF := hatchetwf.NewDunningCommunicationWorkflow(c, dunningSteps)
	dunningResultWF := hatchetwf.NewDunningResultWorkflow(c, dunningSteps)

	w, err := c.NewWorker("getpaidhq-events",
		hatchet.WithWorkflows(
			paymentSuccessWF,
			paymentRefundedWF,
			outgoingWebhookWF,
			billingCycleWF,
			billingCycleRunnerWF,
			orgBillingWF,
			billingSweepWF,
			sendReminderWF,
			dunningAttemptWF,
			dunningRunnerWF,
			dunningCommunicationWF,
			dunningResultWF,
		),
		hatchet.WithSlots(50),
		hatchet.WithDurableSlots(500),
		hatchet.WithLogger(&hatchetLog),
	)
	if err != nil {
		logger.Error("Unable to create Hatchet worker", "err", err.Error())
		panic(err)
	}
	h.worker = w

	// Worker task tracing: the instrumentor registers a global OTel tracer
	// provider whose spans are exported to the Hatchet engine's collector
	// (dashboard trace view). Best-effort — a tracing setup failure must not
	// take billing down with it.
	if cfg.TracingEnabled {
		instrumentor, err := hatchetotel.NewInstrumentor()
		if err != nil {
			logger.Warn("Hatchet tracing requested but instrumentor setup failed; continuing without traces", "err", err.Error())
		} else {
			w.Use(instrumentor.Middleware())
			h.instrumentor = instrumentor
			logger.Info("Hatchet tracing enabled (spans exported to the engine collector)")
		}
	}

	// Run the worker under a cancellable context so Close() can stop it.
	workerCtx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.done = make(chan struct{})
	go func() {
		defer close(h.done)
		if err := w.StartBlocking(workerCtx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("Hatchet worker exited", "err", err.Error())
		}
	}()

	logger.Infof("Hatchet engine initialized with worker")
	return h
}

// Close stops the Hatchet worker by cancelling its run context and waits for
// the worker goroutine to exit (bounded), satisfying io.Closer for graceful
// shutdown.
func (h *Hatchet) Close() error {
	if h.cancel != nil {
		h.cancel()
	}
	if h.done != nil {
		select {
		case <-h.done:
		case <-time.After(10 * time.Second):
			h.logger.Warn("Hatchet worker did not stop within 10s")
		}
	}
	// After the worker stops: flush any buffered spans.
	if h.instrumentor != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.instrumentor.Shutdown(ctx); err != nil {
			h.logger.Warn("Hatchet tracing shutdown failed", "err", err.Error())
		}
	}
	return nil
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
		_, err := h.client.RunNoWait(ctx, "payment-success", hatchetwf.PaymentSuccessInput{PaymentContext: pc},
			hatchet.WithRunMetadata(map[string]string{
				"orgId":     pc.OrgId,
				"orderId":   pc.OrderId,
				"paymentId": pc.Payment.PspId,
			}),
		)
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
		_, err := h.client.RunNoWait(ctx, "payment-refunded", hatchetwf.PaymentRefundedInput{PaymentContext: pc},
			hatchet.WithRunMetadata(map[string]string{
				"orgId":     pc.OrgId,
				"orderId":   pc.OrderId,
				"paymentId": pc.Payment.PspId,
			}),
		)
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
		_, err := h.client.RunNoWait(ctx, "outgoing-webhook", wh,
			hatchet.WithRunMetadata(map[string]string{
				"orgId":                 wh.WebhookSubscription.OrgID,
				"webhookSubscriptionId": wh.WebhookSubscription.Id,
				"eventId":               wh.Event.Id,
			}),
		)
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
	// Two branches under the cron + fan-out billing model:
	//
	//   1. Immediately due (IsDueForBilling == true): the subscription was
	//      activated without an upfront checkout payment — system-charges-now, or a
	//      just-ended trial — so RenewsAt/NextRetryAt/TrialEndsAt is already in the
	//      past. Spawn billing-cycle-runner directly so the first charge is durable
	//      and immediate, rather than waiting for the next sweep tick. This
	//      is the Hatchet analog of Temporal's runner first-iteration charge.
	//
	//   2. Not due (the common checkout-payment case): SetActive recorded the
	//      checkout payment as cycle 1 and set RenewsAt into the future, so there is
	//      nothing to charge now. Stay a no-op — the cron billing-sweep drives the
	//      renewal when RenewsAt falls due.
	//
	// The spawn is keyed BillingRunKey(org, sub, CyclesProcessed) — the exact same
	// run key the sweep uses (billing_sweep.go) — so the activation-spawn and any
	// later sweep-spawn for the same cycle dedup, making the sweep a free backstop.
	if !sub.IsDueForBilling(time.Now().UTC()) {
		h.logger.Debugf("StartSubscriptionWorkflow no-op (not due, cron drives renewal) org=%s sub=%s", sub.OrgId, sub.Id)
		return nil
	}

	if _, err := h.client.RunNoWait(ctx, "billing-cycle-runner", sub,
		hatchet.WithRunKey(hatchetwf.BillingRunKey(sub.OrgId, sub.Id, sub.CyclesProcessed)),
		hatchet.WithRunMetadata(map[string]string{"orgId": sub.OrgId, "subscriptionId": sub.Id}),
	); err != nil {
		return err
	}
	h.logger.Infof("StartSubscriptionWorkflow spawned billing-cycle-runner (immediate first charge) org=%s sub=%s cycle=%d", sub.OrgId, sub.Id, sub.CyclesProcessed)
	return nil
}

func (h Hatchet) UpdateSubscriptionWorkflow(ctx context.Context, updateName string, sub domain.Subscription) error {
	// No-op: the UpdateEventKey was consumed only by the retired per-subscription
	// runner. Under cron + fan-out billing the runner is gone, so there is no
	// durable workflow to feed. Subscription state is persisted by the calling
	// service and observed directly by the next billing-sweep; nothing to push.
	h.logger.Debugf("UpdateSubscriptionWorkflow no-op (runner retired) update=%s org=%s sub=%s", updateName, sub.OrgId, sub.Id)
	return nil
}

func (h Hatchet) CancelSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	// No-op: the CancelEventKey was consumed only by the retired per-subscription
	// runner. A cancelled subscription is simply skipped by the billing-sweep's
	// due query, so there is no durable workflow left to signal.
	h.logger.Debugf("CancelSubscriptionWorkflow no-op (runner retired) org=%s sub=%s", sub.OrgId, sub.Id)
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
func (h Hatchet) StartDunningWorkflow(ctx context.Context, input port.StartDunningWorkflowInput) (string, string, error) {
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
		hatchet.WithRunMetadata(map[string]string{
			"orgId":          input.OrgId,
			"campaignId":     campaignId,
			"subscriptionId": input.SubscriptionId,
			"customerId":     input.CustomerId,
		}),
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
