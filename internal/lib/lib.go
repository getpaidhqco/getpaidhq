package lib

import (
	"go.uber.org/fx"
)

// Module exports dependency
var Module = fx.Options(
	fx.Provide(NewRequestHandler),
	fx.Provide(NewEnv),
	fx.Provide(GetLogger),
	fx.Provide(
		fx.Annotate(
			NewDatabase,
			fx.As(new(Database)),
		),
	),
)
