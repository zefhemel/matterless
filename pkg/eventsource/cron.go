package eventsource

import (
	"github.com/robfig/cron"
	"github.com/zefhemel/matterless/pkg/definition"
)

type CronSource struct {
	cron *cron.Cron
	defs []*definition.CronDef
}

func (cs *CronSource) Start() error {
	cs.cron.Start()
	return nil
}

func (cs *CronSource) Stop() {
	cs.cron.Stop()
}

func (c CronSource) ExtendDefinitions(defs *definition.Definitions) {
}

type cronEvent struct {
	Schedule string `json:"schedule"`
}

func NewCronSource(defs []*definition.CronDef, functionInvokeFunc definition.FunctionInvokeFunc) *CronSource {
	c := cron.New()
	for _, def := range defs {
		// variable capture
		myDef := def
		c.AddFunc(def.Schedule, func() {
			functionInvokeFunc(myDef.Function, &cronEvent{myDef.Schedule})
		})
	}
	return &CronSource{
		cron: c,
		defs: defs,
	}
}

var _ EventSource = &CronSource{}
