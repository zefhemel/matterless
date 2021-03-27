package sandbox

import (
	"context"
	"github.com/zefhemel/matterless/pkg/definition"
)

type LogEntry struct {
	Instance FunctionInstance
	Message  string
}

type ModuleMap map[string]string
type EnvMap map[string]string

type Sandbox interface {
	Function(ctx context.Context, name string, env EnvMap, modulesMap ModuleMap, functionConfig definition.FunctionConfig, code string) (FunctionInstance, error)
	Job(ctx context.Context, name string, env EnvMap, modulesMap ModuleMap, functionConfig definition.FunctionConfig, code string) (JobInstance, error)
	Close()
}

type FunctionInstance interface {
	Name() string
	Invoke(ctx context.Context, event interface{}) (interface{}, error)
}

type JobInstance interface {
	Name() string
	Start(ctx context.Context) (EnvMap, error)
	Stop()
}
