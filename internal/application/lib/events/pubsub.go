package events

type PubSub interface {
	Publish(orgId string, topic string, message interface{}) error
	Subscribe(topic string, handler func(topic string, data []byte)) (Subscription, error)
}

type Payload struct {
	Id    string      `json:"id"`
	OrgId string      `json:"org_id"`
	Topic string      `json:"topic"`
	Data  interface{} `json:"data"`
}

type Subscription interface {
	Unsubscribe() error
}
