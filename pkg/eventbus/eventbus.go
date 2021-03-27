package eventbus

type SubscriptionFunc = func(eventName string, eventData interface{}) (interface{}, error)

type EventBus interface {
	Subscribe(pattern string, subscriptionFunc SubscriptionFunc)
	Unsubscribe(pattern string, subscriptionFunc SubscriptionFunc)
	UnsubscribeAllMatchingPattern(pattern string)
	Publish(eventName string, eventData interface{})
	Call(eventName string, eventData interface{}) (interface{}, error)
}
