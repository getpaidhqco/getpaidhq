package temporal

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"payloop/internal/api/middlewares"
	"payloop/internal/application/services"
	"payloop/internal/domain/workflow"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/infrastructure/workflow/temporal/workflows"
	"payloop/internal/lib"
	"testing"
)

func TestTemporal_StartWorkflow(t *testing.T) {
	ctx := context.Background()
	logger := lib.GetLogger()

	app := fx.New(fx.Options(
		lib.Module,
		services.Module,
		middlewares.Module,
		postgres.Module,
		Module,
	), fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			return logger.GetFxLogger()
		}),
		fx.Invoke(func(temporal workflow.Engine) {
			logger.Info("Starting application")

			_, err := temporal.StartWorkflow(ctx, "payment.success", workflows.WorkflowContext{
				EventId: "test1",
				OrderId: "123",
			})
			assert.Equal(t, err, nil)
			assert.Equal(t, err, nil)
		}),
	))
	app.Start(ctx)
	defer func() {
		app.Stop(ctx)
	}()

}

var event = `{
  "event": "charge.success",
  "data": {
    "id": 4631781627,
    "domain": "test",
    "status": "success",
    "reference": "ord_2sGiYWm3ALfkZgky64Mb5KSTCMN",
    "amount": 10000,
    "message": null,
    "gateway_response": "Successful",
    "paid_at": "2025-01-28T18:19:31.000Z",
    "created_at": "2025-01-28T18:19:19.000Z",
    "channel": "card",
    "currency": "ZAR",
    "ip_address": "41.56.192.220",
    "metadata": {
      "cart_id": "",
      "custom_fields": [
        {
          "display_name": "order_id",
          "variable_name": "Order#",
          "value": "ord_2sGiYWm3ALfkZgky64Mb5KSTCMN"
        }
      ],
      "order_id": "ord_2sGiYWm3ALfkZgky64Mb5KSTCMN",
      "org_id": "mollie"
    },
    "fees_breakdown": null,
    "log": null,
    "fees": 449,
    "fees_split": null,
    "authorization": {
      "authorization_code": "AUTH_enwo0fbqy5",
      "bin": "408408",
      "last4": "4081",
      "exp_month": "12",
      "exp_year": "2030",
      "channel": "card",
      "card_type": "visa ",
      "bank": "TEST BANK",
      "country_code": "ZA",
      "brand": "visa",
      "reusable": true,
      "signature": "SIG_qXaTkp5rShDcqlF2EzYI",
      "account_name": null,
      "receiver_bank_account_number": null,
      "receiver_bank": null
    },
    "customer": {
      "id": 234100681,
      "first_name": null,
      "last_name": null,
      "email": "test@testie.com",
      "customer_code": "CUS_p9l7myr66c9igd7",
      "phone": null,
      "metadata": null,
      "risk_action": "default",
      "international_format_phone": null
    },
    "plan": {},
    "subaccount": {},
    "split": {},
    "order_id": null,
    "paidAt": "2025-01-28T18:19:31.000Z",
    "requested_amount": 10000,
    "pos_transaction_data": null,
    "source": {
      "type": "api",
      "source": "merchant_api",
      "entry_point": "transaction_initialize",
      "identifier": null
    }
  }
}`
