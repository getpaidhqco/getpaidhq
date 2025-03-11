package commands

import (
	"github.com/spf13/cobra"
	"payloop/internal/api/middlewares"
	"payloop/internal/api/routes"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"

	"payloop/internal/lib"
)

// ServeCommand test command
type ServeCommand struct{}

func (s *ServeCommand) Short() string {
	return "serve application"
}

func (s *ServeCommand) Setup(cmd *cobra.Command) {}

func (s *ServeCommand) Run() lib.CommandRunner {
	return func(
		middleware middlewares.Middlewares,
		env lib.Env,
		router lib.RequestHandler,
		route routes.Routes,
		logger logger.Logger,
		queue interfaces.QueueService,
		workflowService interfaces.WorkflowService,
		database lib.Database,
	) {
		middleware.Setup()
		route.Setup()

		logger.Info("Running server")
		if env.ServerPort == "" {
			_ = router.Gin.Run()
		} else {
			_ = router.Gin.Run(":" + env.ServerPort)
		}
	}
}

func NewServeCommand() *ServeCommand {
	return &ServeCommand{}
}
