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
	fx.Provide(NewClerkClient), // Provide the Clerk client as an AuthProvider
)
