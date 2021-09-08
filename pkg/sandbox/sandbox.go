package sandbox

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
)

type RunMode int

const (
	RunModeFunction RunMode = iota
	RunModeJob      RunMode = iota
)

type RuntimeFunctionInstantiator func(ctx context.Context, cfg *config.Config, apiURL string, apiToken string, runMode RunMode, name string, logCallback func(funcName, message string), functionConfig *definition.FunctionConfig, code string) (FunctionInstance, error)

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

type Sandbox struct {
	config          *config.Config
	apiURL          string
	apiToken        string
	ceb             *cluster.ClusterEventBus
	functionWorkers []*FunctionExecutionWorker
	jobWorkers      []*JobExecutionWorker
}

func NewSandbox(cfg *config.Config, apiURL string, apiToken string, ceb *cluster.ClusterEventBus) (*Sandbox, error) {
	s := &Sandbox{
		config:          cfg,
		apiURL:          apiURL,
		apiToken:        apiToken,
		ceb:             ceb,
		functionWorkers: []*FunctionExecutionWorker{},
		jobWorkers:      []*JobExecutionWorker{},
	}

	if !cfg.UseSystemDeno {
		if err := ensureDeno(cfg); err != nil {
			return nil, errors.Wrap(err, "ensure deno")
		}
	}

	return s, nil
}

func (s *Sandbox) LoadFunction(name string, functionConfig *definition.FunctionConfig, code string) error {
	worker, err := NewFunctionExecutionWorker(s.config, s.apiURL, s.apiToken, s.ceb, name, functionConfig, code)
	if err != nil {
		return err
	}
	s.functionWorkers = append(s.functionWorkers, worker)
	return nil
}

func (s *Sandbox) LoadJob(name string, functionConfig *definition.FunctionConfig, code string) error {
	worker, err := NewJobExecutionWorker(s.config, s.apiURL, s.apiToken, s.ceb, name, functionConfig, code)
	if err != nil {
		return err
	}
	s.jobWorkers = append(s.jobWorkers, worker)
	return nil
}

func (s *Sandbox) Flush() {
	for _, worker := range s.functionWorkers {
		if err := worker.Close(); err != nil {
			log.Errorf("Error closing function worker: %s", err)
		}
	}
	s.functionWorkers = []*FunctionExecutionWorker{}

	for _, worker := range s.jobWorkers {
		if err := worker.Close(); err != nil {
			log.Errorf("Error closing job worker: %s", err)
		}
	}
	s.jobWorkers = []*JobExecutionWorker{}
}
