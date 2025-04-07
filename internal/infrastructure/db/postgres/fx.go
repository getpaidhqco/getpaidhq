package postgres

import (
	"go.uber.org/fx"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

// Module exports dependency
var Module = fx.Options(
	RespositoryModules,
	fx.Provide(
		fx.Annotate(
			func(env lib.Env, logger logger.Logger) lib.Database {
				return NewDatabase(env.Get("DATABASE_URL"), logger)
			},
			fx.ResultTags(`name:"primaryDb"`),
		),
	),
	fx.Provide(
		fx.Annotate(
			func(env lib.Env, logger logger.Logger) lib.Database {
				return NewDatabase(env.Get("REPORTING_DATABASE_URL"), logger)
			},
			fx.ResultTags(`name:"reportingDb"`),
		),
	),
)
