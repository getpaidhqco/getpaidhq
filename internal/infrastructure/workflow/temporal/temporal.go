package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	temporal "go.temporal.io/sdk/workflow"
	"log/slog"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/infrastructure/workflow/temporal/workflows"
	"payloop/internal/lib"
)

type Temporal struct {
	logger logger.Logger
	client client.Client
	worker worker.Worker

	orderActivities activities.OrderActivities

	settingRepository repositories.SettingRepository
	pubsub            events.PubSub
}

func NewTemporalEngine(
	logger logger.Logger,
	env lib.Env,
	orderActivities activities.OrderActivities,
	webhookActivities activities.OutgoingWebhookActivities,
	settingRepository repositories.SettingRepository,
	pubsub events.PubSub,
) interfaces.Engine {
	// The client is orderActivities heavyweight object that should be created once per process.
	// Set our Zap logger so that workflows and activities can use it
	logger.Debugf("Connecting to temporal NewLazyClient [%s]", env.TemporalHost)
	c, err := client.NewLazyClient(client.Options{
		HostPort:  env.TemporalHost,
		Namespace: "subscriptions",
		Logger:    NewZapAdapter(lib.GetZapLogger()),
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

func (t Temporal) StartWorkflow(ctx context.Context, id interfaces.WorkflowType, payload interface{}) (interfaces.Result, error) {

	switch id {
	case "payment.success":
		workflowId := lib.GenerateId("payment_success")
		// start workflow
		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: "events",
		}

		// payload is payment_providers.PaymentWebhookContext
		data := interfaces.WorkflowPayload{
			Data: payload,
		}

		we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, workflows.PaymentSuccessWorkflow, data)
		if err != nil {
			t.logger.Error("Unable to execute workflow", "err", err.Error())
			return interfaces.Result{}, err
		}

		var result interfaces.Result
		err = we.Get(ctx, &result)
		if err != nil {
			t.logger.Error("Unable to get workflow result", "err", err.Error())
			return interfaces.Result{}, err
		}
		t.logger.Debug("Finished PaymentSuccessWorkflow workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "result", result)
		return interfaces.Result{
			Success: true,
			Message: "success",
			Payload: result,
		}, nil
	case interfaces.OutgoingWebhook:
		workflowId := lib.GenerateId("webhook_out")
		// start workflow
		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: "events",
		}

		we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, workflows.OutgoingWebhookWorkflow, payload)
		if err != nil {
			t.logger.Error("Unable to execute workflow", "err", err.Error())
			return interfaces.Result{}, err
		}

		var result interfaces.Result
		err = we.Get(ctx, &result)
		if err != nil {
			t.logger.Error("Unable to get workflow result", "err", err.Error())
			return interfaces.Result{}, err
		}
		t.logger.Info("Finished workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "result", result)
		return interfaces.Result{
			Success: true,
			Message: "success",
			Payload: result,
		}, nil
	default:
		return interfaces.Result{}, nil
	}

}

// Starts the long running subscription workflow
func (t Temporal) StartSubscriptionWorkflow(ctx context.Context, subscription entities.Subscription) error {

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
	_, err = t.settingRepository.Create(ctx, entities.Setting{
		OrgId:    subscription.OrgId,
		ParentId: subscription.Id,
		Id:       "temporal-workflow",
		Type:     "workflow.Execution",
		Value:    string(executionBytes),
	})

	return nil
}

func (t Temporal) UpdateSubscriptionWorkflow(ctx context.Context, updateName string, subscription entities.Subscription) error {

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
		return err
	}

	var oldSub entities.Subscription
	err = updateHandle.Get(ctx, &oldSub)
	if err != nil {
		t.logger.Error("Failed to get setting", "error", err)
	}
	return nil
}
func (t Temporal) SignalSubscriptionWorkflow(ctx context.Context, signal string, subscription entities.Subscription, payload interface{}) error {
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
func (t Temporal) getExecution(subscription entities.Subscription) (temporal.Execution, error) {
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
	var eventData events.Payload
	var sub entities.Subscription

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
