package sqs

import (
	"context"
	"github.com/stretchr/testify/assert"
	"payloop/internal/application/lib/events"
	"payloop/internal/lib"
	"testing"
)

func TestSendMessageToSQS(t *testing.T) {
	logger := lib.GetLogger()
	env := lib.NewEnv()
	client := NewSQSFifoClient(logger, env)

	queueUrl := env.Get("SQS_QUEUE_URL")
	if queueUrl == "" {
		t.Fatal("SQS_QUEUE_URL not set")
	}

	err := client.SendMessage(context.TODO(), events.QueueMessage{
		Data: `
{
  "type": "incoming_webhook",
  "data":{
  "psp": "Paystack",
  "data":{
  "id": "evt_slftdehypnoepcwg6lcu6d7squ",
  "type": "payment_captured",
  "version": "1.0.47",
  "created_on": "2025-03-10T09:41:42.519Z",
  "data": {
    "id": "pay_uztze5omf7nuvjzdouoqsos5sy",
    "action_id": "act_xaefwzojcsbulmhxhb4v2d6v4q",
    "amount": 10000,
    "processed_on": "2025-03-10T09:41:42.5078263Z",
    "response_code": "10000",
    "response_summary": "Approved",
    "balances": {
      "total_authorized": 10000,
      "total_voided": 0,
      "available_to_void": 0,
      "total_captured": 10000,
      "available_to_capture": 0,
      "total_refunded": 0,
      "available_to_refund": 10000
    },
    "metadata": {
      "order_id": "ord_2u71IHnb4kaZdIiMiwACdTmFCQ6",
      "org_id": "org_2syb0uTnhuKtQTaLO6EAk1iIUnu",
      "phase": "recurring",
      "subscription_id": "sub_2u71IBDoFkQcXnrFiX2Ye4EQqjU"
    },
    "currency": "USD",
    "processing": {
      "acquirer_transaction_id": "538164880505401759897",
      "acquirer_reference_number": "92398932833923586187322"
    },
    "event_links": {
      "payment": "https://api.sandbox.checkout.com/payments/pay_uztze5omf7nuvjzdouoqsos5sy",
      "payment_actions": "https://api.sandbox.checkout.com/payments/pay_uztze5omf7nuvjzdouoqsos5sy/actions",
      "refund": "https://api.sandbox.checkout.com/payments/pay_uztze5omf7nuvjzdouoqsos5sy/refunds"
    }
  },
  "_links": {
    "self": {
      "href": "https://api.sandbox.checkout.com/workflows/events/evt_slftdehypnoepcwg6lcu6d7squ"
    },
    "subject": {
      "href": "https://api.sandbox.checkout.com/workflows/events/subject/pay_uztze5omf7nuvjzdouoqsos5sy"
    },
    "payment": {
      "href": "https://api.sandbox.checkout.com/payments/pay_uztze5omf7nuvjzdouoqsos5sy"
    },
    "payment_actions": {
      "href": "https://api.sandbox.checkout.com/payments/pay_uztze5omf7nuvjzdouoqsos5sy/actions"
    },
    "refund": {
      "href": "https://api.sandbox.checkout.com/payments/pay_uztze5omf7nuvjzdouoqsos5sy/refunds"
    }
  }
}
}
}

`,
		Type: events.IncomingWebhook,
	})

	assert.NoError(t, err)
}
