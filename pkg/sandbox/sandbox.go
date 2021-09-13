package sandbox

import (
	"context"
	"sync"
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
	"deno":   newDenoFunctionInstance,
	"docker": newDockerFunctionInstance,
}

type RuntimeJobInstantiator func(ctx context.Context, cfg *config.Config, apiURL string, apiToken string, name string, logCallback func(funcName, message string), jobConfig *definition.JobConfig, code string) (JobInstance, error)

var runtimeJobInstantiators = map[string]RuntimeJobInstantiator{
	"deno":   newDenoJobInstance,
	"docker": newDockerJobInstance,
}

const DefaultRuntime = "deno"

type FunctionInstance interface {
	Name() string
	Invoke(ctx context.Context, event interface{}) (interface{}, error)
	LastInvoked() time.Time
	Kill()
}

type JobInstance interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	DidExit() chan error
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

func (s *Sandbox) StartFunctionWorker(name string, functionConfig *definition.FunctionConfig, code string) error {
	worker, err := NewFunctionExecutionWorker(s.config, s.apiURL, s.apiToken, s.ceb, name, functionConfig, code)
	if err != nil {
		return err
	}
	s.functionWorkers = append(s.functionWorkers, worker)
	return nil
}

func (s *Sandbox) StartJobWorker(name definition.FunctionID, jobConfig *definition.JobConfig, code string) error {
	worker, err := NewJobExecutionWorker(s.config, s.apiURL, s.apiToken, s.ceb, string(name), jobConfig, code)
	if err != nil {
		return err
	}
	s.jobWorkers = append(s.jobWorkers, worker)
	go func() {
		<-worker.done
		workers := make([]*JobExecutionWorker, 0, len(s.jobWorkers))
		for _, w := range s.jobWorkers {
			if w != worker {
				workers = append(workers, w)
			}
		}
		s.jobWorkers = workers
	}()
	return nil
}

func (s *Sandbox) Flush() {
	log.Info("Flushing the sandbox")
	var wg sync.WaitGroup
	for _, worker := range s.functionWorkers {
		wg.Add(1)
		worker2 := worker
		go func() {
			if err := worker2.Close(); err != nil {
				log.Errorf("Error closing function worker: %s", err)
			}
			wg.Done()
		}()
	}

	for _, worker := range s.jobWorkers {
		wg.Add(1)
		worker2 := worker
		go func() {
			if err := worker2.Close(); err != nil {
				log.Errorf("Error closing job worker: %s", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	log.Info("Fully flushed")
	s.functionWorkers = []*FunctionExecutionWorker{}
	//s.jobWorkers = []*JobExecutionWorker{}
}

func (s *Sandbox) AppInfo() *cluster.AppInfo {
	si := &cluster.AppInfo{
		FunctionWorkers: map[string]int{},
		JobWorkers:      map[string]int{},
	}
	for _, functionWorker := range s.functionWorkers {
		si.FunctionWorkers[functionWorker.name]++
	}
	for _, jobWorker := range s.jobWorkers {
		si.JobWorkers[jobWorker.name]++
	}
	return si
}
