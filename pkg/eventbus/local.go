package eventbus

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type LocalEventBus struct {
	subscribers   []*subscription
	subscribeLock sync.Mutex
}

type subscription struct {
	pattern   string
	rePattern *regexp.Regexp
	callback  SubscriptionFunc
}

func NewLocalEventBus() *LocalEventBus {
	return &LocalEventBus{
		subscribers: make([]*subscription, 0, 20),
	}
}

func (l *LocalEventBus) Subscribe(pattern string, subscriptionFunc SubscriptionFunc) {
	l.subscribeLock.Lock()
	defer l.subscribeLock.Unlock()
	l.subscribers = append(l.subscribers, &subscription{
		pattern:   pattern,
		rePattern: patternToRegexp(pattern),
		callback:  subscriptionFunc,
	})
}

func (l *LocalEventBus) Unsubscribe(pattern string, subscriptionFunc SubscriptionFunc) {
	l.subscribeLock.Lock()
	defer l.subscribeLock.Unlock()
	newSubscribers := make([]*subscription, 0, len(l.subscribers))
	subscriptionVal := reflect.ValueOf(subscriptionFunc)
	for _, subscriber := range l.subscribers {
		if subscriber.pattern != pattern || reflect.ValueOf(subscriber.callback).Pointer() != subscriptionVal.Pointer() {
			newSubscribers = append(newSubscribers, subscriber)
		}
	}
	l.subscribers = newSubscribers
}

func patternToRegexp(pattern string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf("^%s$", strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", "(.*)")))
}

func (l *LocalEventBus) Publish(eventName string, eventData interface{}) int {
	listenersNotified := 0
	for _, subscriber := range l.subscribers {
		if subscriber.rePattern.MatchString(eventName) {
			subscriber.callback(eventName, eventData)
			listenersNotified++
		}
	}
	return listenersNotified
}

func (l *LocalEventBus) PublishAsync(eventName string, eventData interface{}) int {
	listenersNotified := 0
	for _, subscriber := range l.subscribers {
		if subscriber.rePattern.MatchString(eventName) {
			go subscriber.callback(eventName, eventData)
			listenersNotified++
		}
	}
	return listenersNotified
}

func (l *LocalEventBus) Request(eventName string, eventData map[string]interface{}, timeout time.Duration) (interface{}, error) {
	var responseListener func(string, interface{})

	// Generate an event name for the response
	responseEventName := fmt.Sprintf("resp:%s", uuid.New().String())
	// Inject this as $responseEvent into the eventData
	eventData[ResponseEventKey] = responseEventName

	// Setup a channel to listen to the response
	responseChannel := make(chan interface{})
	stopped := false

	// Subscribe to the response event
	responseListener = func(eventName string, eventData interface{}) {
		// Validate if unexpected happened (e.g. timeout)
		if !stopped {
			responseChannel <- eventData
		}
	}
	l.Subscribe(responseEventName, responseListener)

	// Ensure cleanup
	defer func() {
		// Clean up
		stopped = true
		close(responseChannel)
		l.Unsubscribe(responseEventName, responseListener)
	}()

	// Trigger the request event
	listeners := l.PublishAsync(eventName, eventData)

	if listeners == 0 {
		// Nobody's listening, return immediately
		return nil, NoListenersError
	}

	select {
	case response := <-responseChannel:
		return response, nil
	case <-time.After(timeout):
		return nil, RequestTimeoutError
	}
}

func (l *LocalEventBus) Respond(to interface{}, eventData interface{}) error {
	if toEvent, ok := to.(map[string]interface{}); ok {
		if responseEvent, ok := toEvent[ResponseEventKey]; ok {
			if responseEventName, ok := responseEvent.(string); ok {
				listeners := l.Publish(responseEventName, eventData)
				if listeners == 0 {
					return errors.New("No listeners")
				}
			} else {
				return fmt.Errorf("Response %s not a string: %s", ResponseEventKey, responseEvent)
			}
		} else {
			return fmt.Errorf("No %s specified in event: %+v", ResponseEventKey, to)
		}
	} else {
		return fmt.Errorf("Respond to event is not a map: %+v", to)
	}
	return nil
}
