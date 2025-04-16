package paystack

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"payloop/internal/api/middlewares"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/lib"
	"testing"
)

var event = `{"event":"charge.success","data":{"id":4877064926,"domain":"test","status":"success","reference":"vu1y5uu7dx","amount":10000,"message":null,"gateway_response":"Successful","paid_at":"2025-04-16T05:36:29.000Z","created_at":"2025-04-16T05:36:24.000Z","channel":"card","currency":"ZAR","ip_address":"41.56.158.253","metadata":{"company":"65396d11-8021-4138-8cd1-be3ae783c3e9","purchaseId":"1744781782147b7178254-a3ff-4d60-bbb2-0d1a985315c5","payloop_order_id":"ord_2vnXOub1RQpDh3M5EyqT90k7TKr"},"fees_breakdown":null,"log":null,"fees":449,"fees_split":null,"authorization":{"authorization_code":"AUTH_n3r1is46p9","bin":"408408","last4":"4081","exp_month":"12","exp_year":"2030","channel":"card","card_type":"visa ","bank":"TEST BANK","country_code":"ZA","brand":"visa","reusable":true,"signature":"SIG_U6qTgu328q6dBhAKJMck","account_name":null,"receiver_bank_account_number":null,"receiver_bank":null},"customer":{"id":262780479,"first_name":"","last_name":"","email":"meiringdewet1@gmail.com","customer_code":"CUS_dcaek126f8ma0h6","phone":"","metadata":null,"risk_action":"default","international_format_phone":null},"plan":{},"subaccount":{},"split":{"id":4250561,"name":"Dynamic Split at 1744781783823","split_code":"SPL_tbClzxijl6","formula":{"type":"percentage","bearer_type":"subaccount","bearer_subaccount":1255774,"subaccounts":[{"original_share":500,"fees":449,"share":5,"subaccount_code":"ACCT_9hws2teupa53qxq","id":1255774,"name":"CheckoutJoy","integration":"563712"},{"original_share":9500,"fees":0,"share":95,"subaccount_code":"ACCT_6hqd4hu9xbkfo5n","id":1258571,"name":"ConnectTest","integration":"1388576"}],"integration":0},"shares":{"paystack":449,"subaccounts":[{"amount":51,"original_share":500,"fees":449,"subaccount_code":"ACCT_9hws2teupa53qxq","id":1255774,"integration":"563712"},{"amount":9500,"original_share":9500,"fees":0,"subaccount_code":"ACCT_6hqd4hu9xbkfo5n","id":1258571,"integration":"1388576"}],"integration":0,"original_share":0,"fees":0}},"order_id":null,"paidAt":"2025-04-16T05:36:29.000Z","requested_amount":10000,"pos_transaction_data":null,"source":{"type":"api","source":"merchant_api","entry_point":"transaction_initialize","identifier":null}}}`

func TestPaystack_HandleWebhook(t *testing.T) {
	ctx := context.Background()
	logger := lib.GetLogger()

	app := fx.New(fx.Options(
		lib.Module,
		middlewares.Module,
		postgres.Module,
		Module,
	), fx.Options(
		fx.WithLogger(func() fxevent.Logger {
			return lib.GetFxLogger()
		}),
		fx.Invoke(func(parser WebhookParser) {
			logger.Info("Starting application")

			_, err := parser.ParseWebhook(ctx, []byte(event))
			assert.Equal(t, err, nil)
		}),
	))
	app.Start(ctx)
	defer func() {
		app.Stop(ctx)
	}()

}
