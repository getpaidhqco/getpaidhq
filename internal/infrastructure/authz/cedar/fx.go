package cedar

import (
	"go.uber.org/fx"
)

// Module exports dependency
var Module = fx.Options(
	fx.Provide(NewCedarAuthz),
	fx.Provide(NewCedarMiddleware),
)
