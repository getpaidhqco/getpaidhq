package cedar

import (
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type CedarMiddleware struct {
	handler lib.RequestHandler
	logger  port.Logger
	env     lib.Env
	client  port.Authz
}

func NewCedarMiddleware(handler lib.RequestHandler, logger port.Logger, env lib.Env) CedarMiddleware {

	client := NewCedarAuthz(logger, env)

	return CedarMiddleware{
		handler: handler,
		logger:  logger,
		env:     env,
		client:  client,
	}
}

// Setup sets up cognito middleware
func (m CedarMiddleware) Setup() {
	m.logger.Info("Setting up cedar middleware")

}
