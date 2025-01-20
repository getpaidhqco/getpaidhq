package db

import "go.uber.org/fx"

var Module = fx.Module("db",
	fx.Provide(
		fx.Annotate(
			NewDatabase,
			fx.As(new(Database)),
		),
	),
)
