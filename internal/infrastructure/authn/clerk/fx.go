package clerk

import (
	"go.uber.org/fx"
)

// Module exports dependency
var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewClerkMiddleware,
			fx.ResultTags(`group:"authenticators"`),
		),
	),
)
