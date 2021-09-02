package eventbus

import (
	"github.com/pkg/errors"
)

type SubscriptionFunc = func(eventName string, eventData interface{})

const ResponseEventKey string = "$response_event"

var NoListenersError = errors.New("no listeners for event")
var RequestTimeoutError = errors.New("request timeout")

// The `pattern` mentioned here are event names, plus the support of "*" that matches any substring, e.g.
// The pattern "logs:*" would match any event prefixed with "logs:" such as "logs:myfunc"
