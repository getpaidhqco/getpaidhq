package postgres

import (
	"go.uber.org/fx"
	"payloop/internal/lib"
)

// Module exports dependency
var Module = fx.Options(
	RespositoryModules,
	fx.Provide(
		fx.Annotate(
			NewDatabase,
			fx.As(new(lib.Database)),
		),
	),
)
