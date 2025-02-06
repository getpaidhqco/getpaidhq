package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	temporal "go.temporal.io/sdk/workflow"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/workflow"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/infrastructure/workflow/temporal/workflows"
	"payloop/internal/lib"
)

type Temporal struct {
	logger lib.Logger
	client client.Client
	worker worker.Worker

	orderActivities activities.OrderActivities

	// services
	orderService      services.OrderService
	sessionService    services.SessionService
	settingRepository repositories.SettingRepository
	pubsub            events.PubSub
}

func NewTemporalEngine(
	logger lib.Logger,
	orderService services.OrderService,
	sessionService services.SessionService,
	a activities.OrderActivities,
	settingRepository repositories.SettingRepository,
	pubsub events.PubSub,
) workflow.Engine {
	// The client is a heavyweight object that should be created once per process.
	// Set our Zap logger so that workflows and activities can use it
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
		Logger:   NewZapAdapter(logger.GetZapLogger()),
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
	w.RegisterWorkflow(workflows.SubscriptionWorkflow)

	// Activities

	w.RegisterActivity(&a)

	// Start the worker
	err = w.Start()
	if err != nil {
		logger.Fatalln("Unable to start worker", err)
	}

	logger.Infof("Temporal engine initialized with worker")
	t := Temporal{
		logger:            logger,
		client:            c,
		worker:            w,
		orderService:      orderService,
		sessionService:    sessionService,
		orderActivities:   a,
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

func (t Temporal) StartWorkflow(ctx context.Context, id workflow.WorkflowType, payload interface{}) (workflow.Result, error) {

	switch id {
	case "payment.success":
		workflowId := lib.GenerateId("wf")
		// start workflow
		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: "events",
		}

		// payload is payment_providers.PaymentWebhookContext
		data := workflow.WorkflowPayload{
			Data: payload,
		}

		we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, workflows.PaymentSuccessWorkflow, data)
		if err != nil {
			t.logger.Error("Unable to execute workflow", "err", err.Error())
			return workflow.Result{}, err
		}

		var result workflow.Result
		err = we.Get(ctx, &result)
		if err != nil {
			t.logger.Error("Unable to get workflow result", "err", err.Error())
			return workflow.Result{}, err
		}
		t.logger.Info("Finished workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "result", result)
		return workflow.Result{
			Success: true,
			Message: "success",
			Payload: result,
		}, nil

	default:
		return workflow.Result{}, nil
	}

}

func (t Temporal) StartSubscriptionWorkflow(ctx context.Context, subscription entities.Subscription) (workflow.Result, error) {

	workflowId := fmt.Sprintf(`subscription_[%s]_[%s]`, subscription.OrgId, subscription.Id)
	// start workflow
	// TODO move subscriptions to their own task queue
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowId,
		TaskQueue: "events",
	}
	we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, workflows.SubscriptionWorkflow, subscription)
	if err != nil {
		t.logger.Error("Unable to execute workflow", "err", err.Error())
		return workflow.Result{}, err
	}

	var result workflow.Result
	err = we.Get(ctx, &result)
	if err != nil {
		t.logger.Error("Unable to get workflow result", "err", err.Error())
		return workflow.Result{}, err
	}
	t.logger.Info("Finished workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "result", result)
	return workflow.Result{
		Success: true,
		Message: "success",
		Payload: result,
	}, nil

}

// HandleSubscriptionEvent forwards subscription events on to the appropriate workflow
func (t Temporal) HandleSubscriptionEvent(topic string, data []byte) error {
	// Unmarshal the event data
	var eventData entities.Subscription
	err := json.Unmarshal(data, &eventData)
	if err != nil {
		t.logger.Error("Failed to unmarshal event data", "error", err)
		return err
	}

	switch topic {
	case "subscription.created":
		// TODO this should be done from somewhere else
		t.logger.Infof("Starting subscription workflow [%s][%s]", eventData.OrgId, eventData.Id)
		_, err = t.StartSubscriptionWorkflow(context.TODO(), eventData)

		return err
	default:
		setting, err := t.settingRepository.FindById(context.TODO(), eventData.OrgId, eventData.Id, "temporal-workflow")
		if err != nil {
			t.logger.Error("Failed to get setting", "error", err)
			return err
		}

		var we temporal.Execution
		err = json.Unmarshal([]byte(setting.Value), &we)
		if err != nil {
			t.logger.Error("Failed to unmarshal setting value", "error", err)
			return err
		}

		err = t.client.SignalWorkflow(context.Background(), we.ID, we.RunID, topic, eventData)
		if err != nil {
			t.logger.Error("Unable to signal workflow: %v", err)
		}
		return nil
	}

}
