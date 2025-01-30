package temporal

import (
	"context"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"payloop/internal/application/services"
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
	orderService   services.OrderService
	sessionService services.SessionService
}

func NewTemporalEngine(
	logger lib.Logger,
	orderService services.OrderService,
	sessionService services.SessionService,
	a activities.OrderActivities,
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

	// Activities

	w.RegisterActivity(&a)

	// Start the worker
	err = w.Start()
	if err != nil {
		logger.Fatalln("Unable to start worker", err)
	}

	logger.Infof("Temporal engine initialized with worker")
	return Temporal{
		logger:          logger,
		client:          c,
		worker:          w,
		orderService:    orderService,
		sessionService:  sessionService,
		orderActivities: a,
	}
}

func (t Temporal) StartWorkflow(ctx context.Context, id workflow.WorkflowType, payload interface{}) (workflow.Result, error) {

	switch id {
	case "payment.success":
		workflowId := lib.GenerateId("workflow")
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
