package http

import (
	"go.uber.org/fx"
	stdhttp "net/http"
)

var Module = fx.Module("http",
	fx.Provide(
		fx.Annotate(
			Server,
			fx.As(new(stdhttp.Handler)),
		),
	))
