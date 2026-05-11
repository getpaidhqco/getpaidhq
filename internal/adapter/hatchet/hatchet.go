package hatchet

import (
	"context"
	"encoding/json"
	hatchetwf "payloop/internal/adapter/hatchet/workflows"
	"payloop/internal/adapter/hatchet/steps"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// Hatchet implements port.Engine using Hatchet as the workflow runtime.
//
// Construction order is preserved from the Temporal adapter: narrow services
// produce orderSteps + webhookSteps, those are passed in here, this engine
// builds the workflow definitions + starts the worker, and the engine-aware
// services (SubscriptionOrchestrationService, OrderService, WebhookService)
// take the resulting port.Engine.
type Hatchet struct {
	logger            port.Logger
	client            *hatchet.Client
	worker            *hatchet.Worker
	errorReporter     lib.ErrorReporter
	settingRepository port.SettingRepository
	pubsub            port.PubSub
}

func NewHatchetEngine(
	logger port.Logger,
	env lib.Env,
	orderSteps *steps.OrderSteps,
	errorReporter lib.ErrorReporter,
	webhookSteps *steps.OutgoingWebhookSteps,
	settingRepository port.SettingRepository,
	pubsub port.PubSub,
) port.Engine {
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

	// Build workflows. The runner needs the client so it can spawn child
	// workflows; the other workflows only need their step deps.
	paymentSuccessWF := hatchetwf.NewPaymentSuccessWorkflow(c, orderSteps)
	paymentRefundedWF := hatchetwf.NewPaymentRefundedWorkflow(c, orderSteps)
	outgoingWebhookWF := hatchetwf.NewOutgoingWebhookWorkflow(c, webhookSteps)
	billingCycleWF := hatchetwf.NewBillingCycleWorkflow(c, orderSteps)
	reminderWF := hatchetwf.NewSubscriptionChargeReminderWorkflow(c, orderSteps)
	subscriptionRunnerWF := hatchetwf.NewSubscriptionRunnerWorkflow(c, orderSteps)

	w, err := c.NewWorker("payloop-events",
		hatchet.WithWorkflows(
			paymentSuccessWF,
			paymentRefundedWF,
			outgoingWebhookWF,
			billingCycleWF,
			reminderWF,
			subscriptionRunnerWF,
		),
		hatchet.WithSlots(50),
		hatchet.WithDurableSlots(500),
	)
	if err != nil {
		logger.Error("Unable to create Hatchet worker", "err", err.Error())
		panic(err)
	}

	go func() {
		if err := w.StartBlocking(context.Background()); err != nil {
			logger.Error("Hatchet worker exited", "err", err.Error())
		}
	}()

	logger.Infof("Hatchet engine initialized with worker")

	h := Hatchet{
		logger:            logger,
		client:            c,
		worker:            w,
		errorReporter:     errorReporter,
		settingRepository: settingRepository,
		pubsub:            pubsub,
	}

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

	payload := map[string]string{
		"run_id":        ref.RunId,
		"workflow_name": "subscription-runner",
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = h.settingRepository.Create(ctx, domain.Setting{
		OrgId:    sub.OrgId,
		ParentId: sub.Id,
		Id:       "hatchet-workflow",
		Type:     "hatchet.RunRef",
		Value:    string(b),
	})
	return err
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
