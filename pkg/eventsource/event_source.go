package eventsource

import (
	"github.com/zefhemel/matterless/pkg/definition"
)

type EventSource interface {
	Close()
	ExtendDefinitions(defs *definition.Definitions)
}
