package temporal

import (
	"context"
	"errors"

	enums "go.temporal.io/api/enums/v1"
	serviceerror "go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/adapter/temporal/workflows"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// Temporal implements port.Engine and port.DunningEngine.
//
// All workflow ids are deterministic (workflows/keys.go) so the engine can
// address per-aggregate runners by id without persisting handles in a side
// table. Pubsub fan-in is handled by SubscriptionEventBridge in the service
// layer, not here.
type Temporal struct {
	logger        port.Logger
	client        client.Client
	worker        worker.Worker
	errorReporter lib.ErrorReporter
	taskQueue     string
}

func NewTemporalEngine(
	logger port.Logger,
	env lib.Env,
	orderActivities activities.OrderActivities,
	webhookActivities activities.OutgoingWebhookActivities,
	dunningActivities activities.DunningActivities,
	errorReporter lib.ErrorReporter,
) *Temporal {
	logger.Infof("Initializing Temporal engine [host=%s][namespace=%s][taskQueue=%s]", env.TemporalHost, env.TemporalNamespace, env.TemporalTaskQueue)

	c, err := client.NewLazyClient(client.Options{
		HostPort:  env.TemporalHost,
		Namespace: env.TemporalNamespace,
		Logger:    NewZapAdapter(lib.GetZapLogger()),
	})
	if err != nil {
		logger.Error("Unable to create Temporal client", "err", err.Error())
		panic(err)
	}

	taskQueue := env.TemporalTaskQueue
	if taskQueue == "" {
		taskQueue = "getpaidhq-events"
	}

	w := worker.New(c, taskQueue, worker.Options{})

	// Workflows
	w.RegisterWorkflow(workflows.PaymentSuccessWorkflow)
	w.RegisterWorkflow(workflows.PaymentRefunded)
	w.RegisterWorkflow(workflows.OutgoingWebhookWorkflow)
	w.RegisterWorkflow(workflows.SubscriptionWorkflow)
	w.RegisterWorkflow(workflows.SubscriptionChargeReminder)
	w.RegisterWorkflow(workflows.BillingCycleWorkflow)
	w.RegisterWorkflow(workflows.DunningRunnerWorkflow)
	w.RegisterWorkflow(workflows.DunningAttemptWorkflow)

	// Activities
	w.RegisterActivity(&orderActivities)
	w.RegisterActivity(&webhookActivities)
	w.RegisterActivity(&dunningActivities)

	if err := w.Start(); err != nil {
		logger.Error("Unable to start Temporal worker", "err", err.Error())
		panic(err)
	}

	logger.Infof("Temporal engine initialized with worker")

	return &Temporal{
		logger:        logger,
		client:        c,
		worker:        w,
		errorReporter: errorReporter,
		taskQueue:     taskQueue,
	}
}

// ---- port.Engine ----

func (t *Temporal) StartWorkflow(ctx context.Context, id port.WorkflowType, payload any) (port.WorkflowResult, error) {
	switch id {
	case port.WorkflowPaymentSuccess:
		pc, err := paymentContextFrom(payload)
		if err != nil {
			return port.WorkflowResult{}, err
		}
		opts := client.StartWorkflowOptions{
			ID:        lib.GenerateId("payment_success"),
			TaskQueue: t.taskQueue,
		}
		we, err := t.client.ExecuteWorkflow(ctx, opts, workflows.PaymentSuccessWorkflow, workflows.PaymentSuccessInput{PaymentContext: pc})
		if err != nil {
			t.logger.Error("Unable to execute PaymentSuccessWorkflow", "err", err.Error())
			return port.WorkflowResult{}, err
		}
		return port.WorkflowResult{Success: true, Message: "payment-success queued", Payload: we.GetID()}, nil

	case port.WorkflowPaymentRefunded:
		pc, err := paymentContextFrom(payload)
		if err != nil {
			return port.WorkflowResult{}, err
		}
		opts := client.StartWorkflowOptions{
			ID:        lib.GenerateId("payment_refunded"),
			TaskQueue: t.taskQueue,
		}
		we, err := t.client.ExecuteWorkflow(ctx, opts, workflows.PaymentRefunded, pc)
		if err != nil {
			t.logger.Error("Unable to execute PaymentRefunded", "err", err.Error())
			return port.WorkflowResult{}, err
		}
		return port.WorkflowResult{Success: true, Message: "payment-refunded queued", Payload: we.GetID()}, nil

	case port.WorkflowOutgoingWebhook:
		wh, ok := payload.(port.OutgoingWebhookPayload)
		if !ok {
			return port.WorkflowResult{}, errors.New("outgoing-webhook expects port.OutgoingWebhookPayload")
		}
		opts := client.StartWorkflowOptions{
			ID:        lib.GenerateId("webhook_out"),
			TaskQueue: t.taskQueue,
		}
		we, err := t.client.ExecuteWorkflow(ctx, opts, workflows.OutgoingWebhookWorkflow, wh)
		if err != nil {
			t.logger.Error("Unable to execute OutgoingWebhookWorkflow", "err", err.Error())
			return port.WorkflowResult{}, err
		}
		return port.WorkflowResult{Success: true, Message: "outgoing-webhook queued", Payload: we.GetID()}, nil

	default:
		t.logger.Warnf("Unsupported workflow type: %s", id)
		return port.WorkflowResult{}, nil
	}
}

func (t *Temporal) StartSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	opts := client.StartWorkflowOptions{
		ID:                       workflows.SubscriptionWorkflowID(sub.OrgId, sub.Id),
		TaskQueue:                t.taskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}
	we, err := t.client.ExecuteWorkflow(ctx, opts, workflows.SubscriptionWorkflow, sub)
	if err != nil {
		t.logger.Error("Unable to start SubscriptionWorkflow", "err", err.Error())
		return err
	}
	t.logger.Info("Started SubscriptionWorkflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "OrgId", sub.OrgId, "SubscriptionId", sub.Id)
	return nil
}

func (t *Temporal) UpdateSubscriptionWorkflow(ctx context.Context, updateName string, sub domain.Subscription) error {
	workflowID := workflows.SubscriptionWorkflowID(sub.OrgId, sub.Id)
	t.logger.Debugf("Signaling subscription workflow [%s][%s]", workflowID, updateName)
	if err := t.client.SignalWorkflow(ctx, workflowID, "", updateName, sub); err != nil {
		if isNotFound(err) {
			t.logger.Warnf("Subscription workflow not found for signal [%s][%s]", workflowID, updateName)
			return nil
		}
		t.logger.Error("Failed to signal subscription workflow", "err", err.Error(), "workflowId", workflowID)
		t.errorReporter.ReportError(ctx, err, map[string]any{
			"org_id":          sub.OrgId,
			"workflow_id":     workflowID,
			"update_name":     updateName,
			"subscription_id": sub.Id,
		})
		return err
	}
	return nil
}

func (t *Temporal) CancelSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	workflowID := workflows.SubscriptionWorkflowID(sub.OrgId, sub.Id)
	t.logger.Debugf("Signaling cancel to subscription workflow [%s]", workflowID)
	if err := t.client.SignalWorkflow(ctx, workflowID, "", workflows.SignalCancelRunner, sub); err != nil {
		if isNotFound(err) {
			return nil
		}
		t.logger.Error("Failed to signal subscription workflow cancel", "err", err.Error())
		return err
	}
	return nil
}

func (t *Temporal) SignalSubscriptionWorkflow(ctx context.Context, signal string, sub domain.Subscription, payload any) error {
	workflowID := workflows.SubscriptionWorkflowID(sub.OrgId, sub.Id)
	signalName := signal
	if signal == "webhook-signal" {
		signalName = workflows.WebhookSignalName(sub.OrgId, sub.Id)
	}
	t.logger.Debugf("Signaling subscription workflow [%s][%s]", workflowID, signalName)
	if err := t.client.SignalWorkflow(ctx, workflowID, "", signalName, payload); err != nil {
		if isNotFound(err) {
			return nil
		}
		t.logger.Error("Failed to signal subscription workflow", "err", err.Error(), "signal", signalName)
		return err
	}
	return nil
}

// ---- port.DunningEngine ----

func (t *Temporal) StartDunningWorkflow(ctx context.Context, input domain.StartDunningWorkflowInput) (string, string, error) {
	campaignId := ""
	if input.Metadata != nil {
		campaignId = input.Metadata["campaign_id"]
	}
	if campaignId == "" {
		campaignId = input.SubscriptionId
	}

	workflowID := workflows.DunningWorkflowID(input.OrgId, campaignId)
	opts := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                t.taskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}
	runnerInput := workflows.DunningRunnerInput{
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
	we, err := t.client.ExecuteWorkflow(ctx, opts, workflows.DunningRunnerWorkflow, runnerInput)
	if err != nil {
		t.logger.Error("Unable to start DunningRunnerWorkflow", "err", err.Error())
		return "", "", err
	}
	t.logger.Info("Started DunningRunnerWorkflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "OrgId", input.OrgId, "CampaignId", campaignId)
	return we.GetID(), we.GetRunID(), nil
}

func (t *Temporal) SignalDunningWorkflow(ctx context.Context, signal string, campaign domain.DunningCampaign, payload any) error {
	workflowID := workflows.DunningWorkflowID(campaign.OrgId, campaign.Id)
	signalName := signal
	if signal == "payment_method.updated" {
		signalName = workflows.SignalDunningPaymentMethodUpd
	}
	t.logger.Debugf("Signaling dunning workflow [%s][%s]", workflowID, signalName)
	if err := t.client.SignalWorkflow(ctx, workflowID, "", signalName, payload); err != nil {
		if isNotFound(err) {
			return nil
		}
		t.logger.Error("Failed to signal dunning workflow", "err", err.Error(), "signal", signalName)
		return err
	}
	return nil
}

func (t *Temporal) CancelDunningWorkflow(ctx context.Context, campaign domain.DunningCampaign) error {
	workflowID := workflows.DunningWorkflowID(campaign.OrgId, campaign.Id)
	t.logger.Debugf("Signaling cancel to dunning workflow [%s]", workflowID)
	if err := t.client.SignalWorkflow(ctx, workflowID, "", workflows.SignalDunningCancel, campaign); err != nil {
		if isNotFound(err) {
			return nil
		}
		t.logger.Error("Failed to cancel dunning workflow", "err", err.Error())
		return err
	}
	return nil
}

// ---- helpers ----

func paymentContextFrom(payload any) (domain.PaymentWebhookContext, error) {
	if pc, ok := payload.(domain.PaymentWebhookContext); ok {
		return pc, nil
	}
	return domain.ParsePaymentWebhookContext(payload)
}

func isNotFound(err error) bool {
	var nf *serviceerror.NotFound
	return errors.As(err, &nf)
}
