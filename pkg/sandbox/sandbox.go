package sandbox

import "context"

type LogEntry struct {
	Instance FunctionInstance
	Message  string
}

type ModuleMap map[string]string
type EnvMap map[string]string

type Sandbox interface {
	Function(ctx context.Context, name string, env EnvMap, modulesMap ModuleMap, code string) (FunctionInstance, error)
	Logs() chan LogEntry
	Close()
}

type FunctionInstance interface {
	Name() string
	Invoke(ctx context.Context, event interface{}) (interface{}, error)
}
