package factories

import (
	"context"
	"go.uber.org/fx"
	"payloop/internal/api/middlewares"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/lib"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGatewayFactory_NewGateway(t *testing.T) {
	ctx := context.Background()
	orgId := "mollie"

	app := fx.New(fx.Options(
		lib.Module,
		Module,
		middlewares.Module,
		postgres.Module,
	), fx.Options(
		fx.Invoke(func(gatewayFactory GatewayFactory) {

			gw, err := gatewayFactory.NewGateway(ctx, orgId, "Paystack")
			assert.Equal(t, err, nil)
			assert.NotNil(t, gw, "gateway should not be nil")
		}),
	))
	app.Start(ctx)
	defer func() {
		app.Stop(ctx)
	}()

}
