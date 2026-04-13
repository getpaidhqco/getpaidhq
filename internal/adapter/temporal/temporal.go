package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	temporal "go.temporal.io/sdk/workflow"
	"log/slog"
	"payloop/internal/adapter/temporal/activities"
	"payloop/internal/adapter/temporal/workflows"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type Temporal struct {
	logger          port.Logger
	client          client.Client
	worker          worker.Worker
	errorReporter   lib.ErrorReporter
	orderActivities activities.OrderActivities

	settingRepository port.SettingRepository
	pubsub            port.PubSub
}

func NewTemporalEngine(
	logger port.Logger,
	env lib.Env,
	orderActivities activities.OrderActivities,
	errorReporter lib.ErrorReporter,
	webhookActivities activities.OutgoingWebhookActivities,
	settingRepository port.SettingRepository,
	pubsub port.PubSub,
) port.Engine {
	// The client is orderActivities heavyweight object that should be created once per process.
	// Set our Zap logger so that workflows and activities can use it
	logger.Debugf("Connecting to temporal NewLazyClient [%s]", env.TemporalHost)
	c, err := client.NewLazyClient(client.Options{
		HostPort:  env.TemporalHost,
		Namespace: "subscriptions",
		Logger:    NewSlogAdapter(lib.GetSlogLogger()),
	})
	if err != nil {
		logger.Error("Unable to create client: ", err.Error())
	}

	// Start orderActivities worker and register all workflows and activities for this instance.
	// It's recommended by Temporal to have one worker per process,
	// and to start out with one taskQueue.
	w := worker.New(c, "events", worker.Options{})

	// Workflows
	w.RegisterWorkflow(workflows.PaymentSuccessWorkflow)
	w.RegisterWorkflow(workflows.SubscriptionChargeReminder)
	w.RegisterWorkflow(workflows.SubscriptionWorkflow)
	w.RegisterWorkflow(workflows.OutgoingWebhookWorkflow)
	w.RegisterWorkflow(workflows.PaymentRefunded)

	w.RegisterActivity(&orderActivities)
	w.RegisterActivity(&webhookActivities)

	// Start the worker
	err = w.Start()
	if err != nil {
		panic(err)
	}

	logger.Infof("Temporal engine initialized with worker")
	t := Temporal{
		logger:            logger,
		client:            c,
		errorReporter:     errorReporter,
		worker:            w,
		orderActivities:   orderActivities,
		pubsub:            pubsub,
		settingRepository: settingRepository,
	}

	_, err = pubsub.Subscribe("subscription.*", func(topic string, data []byte) {
		err := t.HandleSubscriptionEvent(topic, data)
		if err != nil {
			logger.Error("Failed to handle subscription event", "error", err.Error())
		}
	})
	if err != nil {
		logger.Error("Failed to subscribe to subscription paused topic", "error", err.Error())
		panic(err)
	}

	return t
}

func (t Temporal) StartWorkflow(ctx context.Context, id port.WorkflowType, payload interface{}) (port.WorkflowResult, error) {

	switch id {
	case "payment.success":
		workflowId := lib.GenerateId("payment_success")
		// start workflow
		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: "events",
		}

		// payload is domain.PaymentWebhookContext
		data := port.WorkflowPayload{
			Data: payload,
		}

		we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, workflows.PaymentSuccessWorkflow, data)
		if err != nil {
			t.logger.Error("Unable to execute workflow", "err", err.Error())
			return port.WorkflowResult{}, err
		}

		var result port.WorkflowResult
		err = we.Get(ctx, &result)
		if err != nil {
			t.logger.Error("Unable to get workflow result", "err", err.Error())
			return port.WorkflowResult{}, err
		}
		t.logger.Debug("Finished PaymentSuccessWorkflow workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "result", result)
		return port.WorkflowResult{
			Success: true,
			Message: "success",
			Payload: result,
		}, nil
	case port.WorkflowOutgoingWebhook:
		workflowId := lib.GenerateId("webhook_out")
		// start workflow
		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: "events",
		}

		we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, workflows.OutgoingWebhookWorkflow, payload)
		if err != nil {
			t.logger.Error("Unable to execute workflow", "err", err.Error())
			return port.WorkflowResult{}, err
		}

		var result port.WorkflowResult
		err = we.Get(ctx, &result)
		if err != nil {
			t.logger.Error("Unable to get workflow result", "err", err.Error())
			return port.WorkflowResult{}, err
		}
		t.logger.Info("Finished workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "result", result)
		return port.WorkflowResult{
			Success: true,
			Message: "success",
			Payload: result,
		}, nil

	case port.WorkflowPaymentRefunded:
		workflowId := lib.GenerateId("refund")
		// start workflow
		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: "events",
		}

		we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, workflows.PaymentRefunded, payload)
		if err != nil {
			t.logger.Error("Unable to execute workflow", "err", err.Error())
			return port.WorkflowResult{}, err
		}

		var result port.WorkflowResult
		err = we.Get(ctx, &result)
		if err != nil {
			t.logger.Error("Unable to get workflow result", "err", err.Error())
			return port.WorkflowResult{}, err
		}
		t.logger.Info("Finished workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "result", result)
		return port.WorkflowResult{
			Success: true,
			Message: "success",
			Payload: result,
		}, nil
	default:
		t.logger.Warnf("Unsupported workflow type: %s", id)
		return port.WorkflowResult{}, nil
	}

}

// Starts the long running subscription workflow
func (t Temporal) StartSubscriptionWorkflow(ctx context.Context, subscription domain.Subscription) error {

	workflowId := fmt.Sprintf(`sub_[%s]_[%s]`, subscription.OrgId, subscription.Id)
	// start workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowId,
		TaskQueue: "events",
	}
	we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, workflows.SubscriptionWorkflow, subscription)
	if err != nil {
		t.logger.Error("Unable to execute workflow", "err", err.Error())
		return err
	}
	t.logger.Info("Finished workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID())
	executionBytes, err := json.Marshal(temporal.Execution{
		ID:    we.GetID(),
		RunID: we.GetRunID(),
	})
	if err != nil {
		return err
	}
	_, err = t.settingRepository.Create(ctx, domain.Setting{
		OrgId:    subscription.OrgId,
		ParentId: subscription.Id,
		Id:       "temporal-workflow",
		Type:     "workflow.Execution",
		Value:    string(executionBytes),
	})

	return nil
}

func (t Temporal) UpdateSubscriptionWorkflow(ctx context.Context, updateName string, subscription domain.Subscription) error {
	we, err := t.getExecution(subscription)
	if err != nil {
		return err
	}

	updateHandle, err := t.client.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   we.ID,
		RunID:        we.RunID,
		UpdateName:   updateName,
		WaitForStage: client.WorkflowUpdateStageCompleted,
		Args:         []interface{}{subscription},
	})
	if err != nil {
		t.logger.Error("Failed to update workflow", "error", slog.String("err", err.Error()))
		t.errorReporter.ReportError(ctx, err, map[string]interface{}{
			"org_id":          subscription.OrgId,
			"workflow_id":     we.ID,
			"run_id":          we.RunID,
			"update_name":     updateName,
			"subscription_id": subscription.Id,
		})
		return err
	}

	var oldSub domain.Subscription
	err = updateHandle.Get(ctx, &oldSub)
	if err != nil {
		t.logger.Error("Failed to get setting", "error", err)
	}
	return nil
}

func (t Temporal) CancelSubscriptionWorkflow(ctx context.Context, subscription domain.Subscription) error {
	we, err := t.getExecution(subscription)
	if err != nil {
		return err
	}

	cancelErr := t.client.CancelWorkflow(ctx, we.ID, we.RunID)
	if cancelErr != nil {
		t.logger.Error("Failed to cancel workflow", "error", slog.String("err", cancelErr.Error()))
		t.errorReporter.ReportError(ctx, err, map[string]interface{}{
			"org_id":          subscription.OrgId,
			"workflow_id":     we.ID,
			"run_id":          we.RunID,
			"subscription_id": subscription.Id,
		})
		return cancelErr
	}

	return nil
}

func (t Temporal) SignalSubscriptionWorkflow(ctx context.Context, signal string, subscription domain.Subscription, payload interface{}) error {
	we, err := t.getExecution(subscription)
	if err != nil {
		t.logger.Error("Failed to get subscription workflow", "error", err)
		return err
	}

	t.logger.Debugf("Signaling workflow [%s][%s]", we.ID, signal)
	err = t.client.SignalWorkflow(ctx, we.ID, we.RunID, signal, payload)
	if err != nil {
		t.logger.Error("Failed to signal workflow", "error", slog.String("err", err.Error()))
		return err
	}
	return nil
}

// HandleSubscriptionEvent forwards subscription events on to the appropriate workflow
func (t Temporal) getExecution(subscription domain.Subscription) (temporal.Execution, error) {
	setting, err := t.settingRepository.FindById(context.TODO(), subscription.OrgId, subscription.Id, "temporal-workflow")
	if err != nil {
		t.logger.Error("Failed to get setting", "error", err)
		return temporal.Execution{}, err
	}

	var we temporal.Execution
	err = json.Unmarshal([]byte(setting.Value), &we)
	if err != nil {
		t.logger.Error("Failed to unmarshal setting value", "error", err)
		return temporal.Execution{}, err
	}

	t.logger.Debugf(`Getting the latest runID for workflow [%s]`, we.ID)
	workflowRun := t.client.GetWorkflow(context.Background(), we.ID, "")
	we.RunID = workflowRun.GetRunID()
	t.logger.Debugf(`Found RunID [%s]`, we.RunID)

	return we, nil
}

// HandleSubscriptionEvent forwards subscription events on to the appropriate workflow
func (t Temporal) HandleSubscriptionEvent(topic string, data []byte) error {
	t.logger.Infof("Received topic [%s]", topic)
	// Unmarshal the event data
	var eventData port.PubSubPayload
	var sub domain.Subscription

	err := json.Unmarshal(data, &eventData)
	if err != nil {
		t.logger.Error("Failed to unmarshal event data", "error", err)
		return err
	}

	dataBytes, err := json.Marshal(eventData.Data)
	if err != nil {
		t.logger.Error("Failed to marshal subscription data", "error", err)
		return err
	}
	err = json.Unmarshal(dataBytes, &sub)
	if err != nil {
		t.logger.Error("Failed to unmarshal event data to Subscription", "error", err)
		return err
	}

	switch topic {
	case "subscription.paused":
		we, err := t.getExecution(sub)
		if err != nil {
			return err
		}

		err = t.client.SignalWorkflow(context.Background(), we.ID, we.RunID, topic, sub)
		if err != nil {
			t.logger.Error("Unable to signal workflow: %v", slog.String("err", err.Error()))
		}
		return nil
	default:
		t.logger.Infof("No handler for topic %s", topic)
	}
	return nil
}
