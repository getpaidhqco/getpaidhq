package events

type PubSub interface {
	Publish(topic string, message string) error
	PublishJSON(topic string, message interface{}) error
}

type Subscription interface {
	Unsubscribe() error
}

const (
	TopicPaymentChargeSuccess = "payment.charge.success"
	TopicChargeFailed         = "charge.failed"
	TopicTransferSuccess      = "transfer.success"
	TopicTransferFailed       = "transfer.failed"

	TopicOrderCompleted = "order.completed"
)
