package eventsource

import (
	"github.com/robfig/cron"
	"github.com/zefhemel/matterless/pkg/definition"
)

type CronSource struct {
	cron *cron.Cron
	defs []*definition.CronDef
}

var _ EventSource = &CronSource{}

func NewCronSource(defs []*definition.CronDef, functionInvokeFunc definition.FunctionInvokeFunc) *CronSource {
	c := cron.New()
	for _, def := range defs {
		// variable capture
		myDef := def
		c.AddFunc(def.Schedule, func() {
			functionInvokeFunc(myDef.Function, &cronEvent{myDef.Schedule})
		})
	}
	c.Start()
	return &CronSource{
		cron: c,
		defs: defs,
	}
}

func (cs *CronSource) Close() {
	cs.cron.Stop()
}

func (c CronSource) ExtendDefinitions(defs *definition.Definitions) {
}

type cronEvent struct {
	Schedule string `json:"schedule"`
}
