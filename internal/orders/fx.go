package orders

import (
	"go.uber.org/fx"
)

var Module = fx.Module("orders",
	fx.Provide(
		fx.Annotate(
			NewOrderRepository,
			fx.As(new(OrderRepository)),
		)),
)
