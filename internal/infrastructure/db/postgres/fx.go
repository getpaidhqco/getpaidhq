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
				return NewDatabase(env.Get("GPHQ_DATABASE_URL"), logger)
			},
			fx.ResultTags(`name:"primaryDb"`),
		),
	),
	fx.Provide(
		fx.Annotate(
			func(env lib.Env, logger logger.Logger) lib.Database {
				return NewDatabase(env.Get("GPHQ_REPORTING_DATABASE_URL"), logger)
			},
			fx.ResultTags(`name:"reportingDb"`),
		),
	),
	fx.Provide(
		fx.Annotate(
			func(env lib.Env, logger logger.Logger) lib.Database {
				return NewDatabase(env.Get("GPHQ_USAGE_DATABASE_URL"), logger)
			},
			fx.ResultTags(`name:"usageDb"`),
		),
	),
	fx.Provide(
		fx.Annotate(
			func(env lib.Env, logger logger.Logger) CdcStream {
				return NewCdcStream(env.Get("GPHQ_DATABASE_URL"), logger)
			},
		),
	),
)
