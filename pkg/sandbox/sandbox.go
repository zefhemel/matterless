package sandbox

import (
	"context"
	"github.com/zefhemel/matterless/pkg/eventbus"
)

type LogEntry struct {
	Instance FunctionInstance
	Message  string
}

type ModuleMap map[string]string
type EnvMap map[string]string

type Sandbox interface {
	Function(ctx context.Context, name string, env EnvMap, modulesMap ModuleMap, code string) (FunctionInstance, error)
	Job(ctx context.Context, name string, env EnvMap, modulesMap ModuleMap, code string) (JobInstance, error)
	EventBus() eventbus.EventBus
	Close()
}

type FunctionInstance interface {
	Name() string
	Invoke(ctx context.Context, event interface{}) (interface{}, error)
}

type JobInstance interface {
	Name() string
	Start(ctx context.Context, params map[string]interface{}) (EnvMap, error)
	Stop()
}
