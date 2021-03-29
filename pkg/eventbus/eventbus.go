package eventbus

import "time"

type SubscriptionFunc = func(eventName string, eventData interface{})

const ResponseEventKey string = "$response_event"

type EventBus interface {
	Subscribe(pattern string, subscriptionFunc SubscriptionFunc)
	Unsubscribe(pattern string, subscriptionFunc SubscriptionFunc)
	UnsubscribeAllMatchingPattern(pattern string)
	Publish(eventName string, eventData interface{})
	PublishAsync(eventName string, eventData interface{})
	Request(eventName string, eventData map[string]interface{}, timeout time.Duration) (interface{}, error)
	Respond(to interface{}, eventData interface{})
}
