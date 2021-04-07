package eventbus

import (
	"github.com/pkg/errors"
	"time"
)

type SubscriptionFunc = func(eventName string, eventData interface{})

const ResponseEventKey string = "$response_event"

var NoListenersError = errors.New("no listeners for event")
var RequestTimeoutError = errors.New("request timeout")

// The `pattern` mentioned here are event names, plus the support of "*" that matches any substring, e.g.
// The pattern "logs:*" would match any event prefixed with "logs:" such as "logs:myfunc"

type EventBus interface {
	Subscribe(pattern string, subscriptionFunc SubscriptionFunc)
	Unsubscribe(pattern string, subscriptionFunc SubscriptionFunc)
	Publish(eventName string, eventData interface{}) int
	PublishAsync(eventName string, eventData interface{}) int
	Request(eventName string, eventData map[string]interface{}, timeout time.Duration) (interface{}, error)
	Respond(to interface{}, eventData interface{}) error
}
