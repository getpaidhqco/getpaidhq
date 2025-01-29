package temporal

import (
	"context"
	"go.temporal.io/sdk/client"
	"log"
	"payloop/internal/domain/workflow"
	"payloop/internal/lib"
)

type Temporal struct {
	logger lib.Logger
	client client.Client
}

func NewTemporalEngine(logger lib.Logger) workflow.Engine {
	// The client is a heavyweight object that should be created once per process.
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	return Temporal{
		logger: logger,
		client: c,
	}
}

func (t Temporal) StartWorkflow(ctx context.Context, id string, payload interface{}) error {
	workflowId := lib.GenerateId("workflow")
	// start workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowId,
		TaskQueue: id,
	}

	we, err := t.client.ExecuteWorkflow(ctx, workflowOptions, PaymentSuccessWorkflow, payload)
	if err != nil {
		t.logger.Error("Unable to execute workflow", "err", err.Error())
		return err
	}
	t.logger.Info("Started workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID())

	var result string
	err = we.Get(ctx, &result)
	if err != nil {
		t.logger.Error("Unable to get workflow result", "err", err.Error())
	}
	t.logger.Info("Finished workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID(), "result", result)
	return nil
}
