package eventsource

import (
	"github.com/robfig/cron"
	"github.com/zefhemel/matterless/pkg/definition"
)

type CronSource struct {
	cron *cron.Cron
	def  *definition.CronDef
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

func NewCronSource(def *definition.CronDef, functionInvokeFunc definition.FunctionInvokeFunc) *CronSource {
	c := cron.New()
	c.AddFunc(def.Schedule, func() {
		functionInvokeFunc(def.Function, &cronEvent{def.Schedule})
	})
	return &CronSource{
		cron: c,
		def:  def,
	}
}

var _ EventSource = &CronSource{}
