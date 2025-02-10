package cedar

import (
	"payloop/internal/application/lib/authz"
	"payloop/internal/lib"
)

type CedarMiddleware struct {
	handler lib.RequestHandler
	logger  lib.Logger
	env     lib.Env
	client  authz.Authz
}

func NewCedarMiddleware(handler lib.RequestHandler, logger lib.Logger, env lib.Env) CedarMiddleware {

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
