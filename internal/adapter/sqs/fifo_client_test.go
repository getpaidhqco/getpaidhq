package sqs

import (
	"github.com/stretchr/testify/assert"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"testing"
	"time"
)

func TestSendMessageToSQS(t *testing.T) {
	logger := lib.GetLogger()
	env := lib.NewEnv()
	client := NewSQSFifoClient(logger, env)

	queueUrl := env.Get("SQS_QUEUE_URL")
	if queueUrl == "" {
		t.Fatal("SQS_QUEUE_URL not set")
	}

	err := client.SendMessage(t.Context(), port.QueueMessage{
		Data: port.PaymentWebhookPayload{
			Psp:  "Paystack",
			Data: "{\"event\":\"charge.success\",\"data\":{\"id\":" + time.Now().String() + ",\"domain\":\"test\",\"status\":\"success\",\"reference\":\"yj2047rw49yh5ff\",\"amount\":100,\"message\":null,\"gateway_response\":\"Approved\",\"paid_at\":\"2025-03-11T13:02:33.000Z\",\"created_at\":\"2025-03-11T13:02:32.000Z\",\"channel\":\"card\",\"currency\":\"ZAR\",\"ip_address\":null,\"metadata\":{\"order_id\":\"ord_2uAiMWxwKiiHIq2Anqgv9wo4hNx\",\"org_id\":\"org_2syb0uTnhuKtQTaLO6EAk1iIUnu\",\"type\":\"recurring\"},\"fees_breakdown\":null,\"log\":null,\"fees\":4,\"fees_split\":null,\"authorization\":{\"authorization_code\":\"AUTH_cdoksiofri\",\"bin\":\"408408\",\"last4\":\"4081\",\"exp_month\":\"12\",\"exp_year\":\"2030\",\"channel\":\"card\",\"card_type\":\"visa \",\"bank\":\"TEST BANK\",\"country_code\":\"ZA\",\"brand\":\"visa\",\"reusable\":true,\"signature\":\"SIG_U6qTgu328q6dBhAKJMck\",\"account_name\":null,\"receiver_bank_account_number\":null,\"receiver_bank\":null},\"customer\":{\"id\":244548544,\"first_name\":null,\"last_name\":null,\"email\":\"test+1@checkoutjoy.com\",\"customer_code\":\"CUS_nlqug0yx4o3db4y\",\"phone\":null,\"metadata\":null,\"risk_action\":\"default\",\"international_format_phone\":null},\"plan\":{},\"subaccount\":{},\"split\":{\"id\":3974260,\"name\":\"Dynamic Split at 1741698152228\",\"split_code\":\"SPL_6wHarzX9iB\",\"formula\":{\"type\":\"percentage\",\"bearer_type\":\"subaccount\",\"bearer_subaccount\":1255774,\"subaccounts\":[{\"original_share\":90,\"fees\":0,\"share\":90,\"subaccount_code\":\"ACCT_6hqd4hu9xbkfo5n\",\"id\":1258571,\"name\":\"ConnectTest\",\"integration\":\"1388576\"},{\"original_share\":10,\"fees\":4,\"share\":10,\"subaccount_code\":\"ACCT_9hws2teupa53qxq\",\"id\":1255774,\"name\":\"CheckoutJoy\",\"integration\":\"563712\"}],\"integration\":0},\"shares\":{\"paystack\":4,\"subaccounts\":[{\"amount\":90,\"original_share\":90,\"fees\":0,\"subaccount_code\":\"ACCT_6hqd4hu9xbkfo5n\",\"id\":1258571,\"integration\":\"1388576\"},{\"amount\":6,\"original_share\":10,\"fees\":4,\"subaccount_code\":\"ACCT_9hws2teupa53qxq\",\"id\":1255774,\"integration\":\"563712\"}],\"integration\":0,\"original_share\":0,\"fees\":0}},\"order_id\":null,\"paidAt\":\"2025-03-11T13:02:33.000Z\",\"requested_amount\":100,\"pos_transaction_data\":null,\"source\":{\"type\":\"api\",\"source\":\"merchant_api\",\"entry_point\":\"charge\",\"identifier\":null}}}",
		},
		Type: port.QueueIncomingWebhook,
	})

	assert.NoError(t, err)
}
