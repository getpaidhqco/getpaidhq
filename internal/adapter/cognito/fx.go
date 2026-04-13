package cognito

import (
	"go.uber.org/fx"
)

// Module exports dependency
var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewCognitoMiddleware,
			fx.ResultTags(`group:"authenticators"`),
		),
	),
)
