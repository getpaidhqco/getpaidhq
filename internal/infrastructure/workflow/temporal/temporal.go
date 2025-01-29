package temporal

import (
	"context"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"log"
	"payloop/internal/domain/workflow"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/infrastructure/workflow/temporal/workflows"
	"payloop/internal/lib"
)

type Temporal struct {
	logger                 lib.Logger
	client                 client.Client
	worker                 worker.Worker
	paymentSuccessWorkflow workflows.PaymentSuccessWorkflow
}

func NewTemporalEngine(
	logger lib.Logger,
) workflow.Engine {
	// The client is a heavyweight object that should be created once per process.
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		logger.Error("Unable to create client: ", err.Error())
	}
	logger.Infof("Temporal engine initialized")

	wf := workflows.NewPaymentSuccessWorkflow(logger)

	w := worker.New(c, "events", worker.Options{})
	w.RegisterWorkflow(wf.Start)
	w.RegisterActivity(activities.CompleteOrder)
	err = w.Start()
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}

	logger.Infof("One worker initialized")
	return Temporal{
		logger:                 logger,
		client:                 c,
		worker:                 w,
		paymentSuccessWorkflow: wf,
	}
}

func (t Temporal) StartWorkflow(ctx context.Context, id string, payload interface{}) (workflow.Result, error) {
	workflowId := lib.GenerateId("workflow")
	// start workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowId,
		TaskQueue: "events",
	}

	we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, t.paymentSuccessWorkflow.Start, payload)
	if err != nil {
		t.logger.Error("Unable to execute workflow", "err", err.Error())
		return workflow.Result{}, err
	}
	t.logger.Info("Started workflow ", "WorkflowID: ", we.GetID(), "RunID: ", we.GetRunID())

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
