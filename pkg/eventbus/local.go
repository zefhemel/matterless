package eventbus

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
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

func (l *LocalEventBus) Call(eventName string, eventData interface{}) (interface{}, error) {
	for _, subscriber := range l.subscribers {
		if subscriber.rePattern.MatchString(eventName) {
			return subscriber.callback(eventName, eventData)
		}
	}
	return nil, errors.New("no listener")
}
