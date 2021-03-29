package eventbus

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
)

type LocalEventBus struct {
	subscribers   []*subscription
	subscribeLock sync.Mutex
}

var _ EventBus = &LocalEventBus{}

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

func (l *LocalEventBus) UnsubscribeAllMatchingPattern(pattern string) {
	l.subscribeLock.Lock()
	defer l.subscribeLock.Unlock()
	newSubscribers := make([]*subscription, 0, len(l.subscribers))
	rePattern := patternToRegexp(pattern)
	for _, subscriber := range l.subscribers {
		if !rePattern.MatchString(subscriber.pattern) {
			newSubscribers = append(newSubscribers, subscriber)
		}
	}
	l.subscribers = newSubscribers
}

func patternToRegexp(pattern string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf("^%s$", strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", "(.*)")))
}

func (l *LocalEventBus) Publish(eventName string, eventData interface{}) {
	for _, subscriber := range l.subscribers {
		if subscriber.rePattern.MatchString(eventName) {
			subscriber.callback(eventName, eventData)
		}
	}
}

func (l *LocalEventBus) PublishAsync(eventName string, eventData interface{}) {
	go l.Publish(eventName, eventData)
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
	defer func() {
		// Clean up
		stopped = true
		close(responseChannel)
	}()

	// Subscribe to the response event
	responseListener = func(eventName string, eventData interface{}) {
		// Check if unexpected happened (e.g. timeout)
		if !stopped {
			responseChannel <- eventData
		}
		l.Unsubscribe(responseEventName, responseListener)
	}
	l.Subscribe(responseEventName, responseListener)

	// Trigger the request event
	l.PublishAsync(eventName, eventData)

	select {
	case response := <-responseChannel:
		return response, nil
	case <-time.After(timeout):
		return nil, errors.New("timeout")
	}
}

func (l *LocalEventBus) Respond(to interface{}, eventData interface{}) {
	if toEvent, ok := to.(map[string]interface{}); ok {
		if responseEvent, ok := toEvent[ResponseEventKey]; ok {
			if responseEventName, ok := responseEvent.(string); ok {
				l.Publish(responseEventName, eventData)
			} else {
				log.Errorf("Response %s not a string: %s", ResponseEventKey, responseEvent)
			}
		} else {
			log.Errorf("No %s specified in event: %+v", ResponseEventKey, to)
		}
	} else {
		log.Errorf("Respond to event is not a map: %+v", to)
	}
}
