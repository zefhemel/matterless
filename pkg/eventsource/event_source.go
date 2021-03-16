package eventsource

import (
	"github.com/zefhemel/matterless/pkg/definition"
)

type EventSource interface {
	Start() error
	Stop()
	ExtendDefinitions(defs *definition.Definitions)
}
