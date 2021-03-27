package eventsource

import (
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
)

type subscription struct {
	eventPattern     string
	subscriptionFunc eventbus.SubscriptionFunc
}

type EventEventSource struct {
	eventBus      eventbus.EventBus
	subscriptions []subscription
}

func NewEventEventSource(eventBus eventbus.EventBus, defs map[string][]definition.FunctionID, functionInvokeFunc definition.FunctionInvokeFunc) *EventEventSource {
	subscriptions := make([]subscription, 0, 10)
	for eventName, funcs := range defs {
		funcsToInvoke := funcs
		sub := subscription{
			eventPattern: eventName,
			subscriptionFunc: func(eventName string, eventData interface{}) (interface{}, error) {
				for _, funcToInvoke := range funcsToInvoke {
					functionInvokeFunc(funcToInvoke, eventData)
				}
				return nil, nil
			},
		}
		eventBus.Subscribe(eventName, sub.subscriptionFunc)
		subscriptions = append(subscriptions, sub)
	}
	return &EventEventSource{
		eventBus:      eventBus,
		subscriptions: subscriptions,
	}
}

func (e *EventEventSource) Close() {
	for _, subscription := range e.subscriptions {
		e.eventBus.Unsubscribe(subscription.eventPattern, subscription.subscriptionFunc)
	}
}

func (e *EventEventSource) ExtendDefinitions(defs *definition.Definitions) {
}

var _ EventSource = &EventEventSource{}
