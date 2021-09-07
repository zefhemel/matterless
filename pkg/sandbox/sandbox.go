package sandbox

import (
	"context"
	"time"

	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
)

type RunMode int

const (
	RunModeFunction RunMode = iota
	RunModeJob      RunMode = iota
)

type RuntimeFunctionInstantiator func(ctx context.Context, config *config.Config, apiURL string, apiToken string, runMode RunMode, name string, logCallback func(funcName, message string), functionConfig *definition.FunctionConfig, code string) (FunctionInstance, error)

var runtimeFunctionInstantiators = map[string]RuntimeFunctionInstantiator{
	"deno": newDenoFunctionInstance,
	"node": newDockerFunctionInstance,
}

type RuntimeJobInstantiator func(ctx context.Context, cfg *config.Config, apiURL string, apiToken string, name string, logCallback func(funcName, message string), functionConfig *definition.FunctionConfig, code string) (JobInstance, error)

var runtimeJobInstantiators = map[string]RuntimeJobInstantiator{
	"deno": newDenoJobInstance,
	"node": newDockerJobInstance,
}

const DefaultRuntime = "deno"

type FunctionInstance interface {
	Name() string
	Invoke(ctx context.Context, event interface{}) (interface{}, error)
	LastInvoked() time.Time
	Kill() error
}

type JobInstance interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
