package commands

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"payloop/internal/api/middlewares"
	"payloop/internal/api/routes"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
	"payloop/internal/mcp"
)

// ServeCommand test command
type ServeCommand struct{}

func (s *ServeCommand) Short() string {
	return "serve application"
}

func (s *ServeCommand) Setup(cmd *cobra.Command) {}

func (s *ServeCommand) Run() lib.CommandRunner {
	return func(
		params struct {
			fx.In
			Middlewares     middlewares.Middlewares
			Env             lib.Env
			Mcp             mcp.MCPServer
			Router          lib.RequestHandler
			Route           routes.Routes
			Logger          logger.Logger
			Queue           interfaces.QueueService
			WorkflowService interfaces.WorkflowService
			PrimaryDb       lib.Database `name:"primaryDb"`
			ReportingDb     lib.Database `name:"reportingDb"`
			Reporter        lib.ErrorReporter
		},
	) {

		params.Middlewares.Setup()
		params.Route.Setup()
		_ = params.Mcp.SSEServer.Start(":8084")
		params.Logger.Info("Running server")
		if params.Env.ServerPort == "" {
			_ = params.Router.Gin.Run()
		} else {
			_ = params.Router.Gin.Run(":" + params.Env.ServerPort)
		}

	}
}

func NewServeCommand() *ServeCommand {
	return &ServeCommand{}
}
