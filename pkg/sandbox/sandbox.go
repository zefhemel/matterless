package sandbox

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"sync"
	"time"
)

type LogEntry struct {
	FunctionName string
	Message      string
}

type RunMode int

const (
	RunModeFunction RunMode = iota
	RunModeJob      RunMode = iota
)

type RuntimeFunctionInstantiator func(ctx context.Context, config *config.Config, apiURL string, apiToken string, runMode RunMode, name string, eventBus eventbus.EventBus, functionConfig *definition.FunctionConfig, code string) (FunctionInstance, error)

var runtimeFunctionInstantiators = map[string]RuntimeFunctionInstantiator{
	"deno": newDenoFunctionInstance,
	"node": newDockerFunctionInstance,
}

type RuntimeJobInstantiator func(ctx context.Context, cfg *config.Config, apiURL string, apiToken string, name string, eventBus eventbus.EventBus, functionConfig *definition.FunctionConfig, code string) (JobInstance, error)

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
	Start() error
	Stop() error
}

type Sandbox struct {
	config                  *config.Config
	runningFunctions        map[string]FunctionInstance
	runningJobs             map[string]JobInstance
	ticker                  *time.Ticker
	done                    chan struct{}
	internalStateUpdateLock sync.RWMutex
	eventBus                eventbus.EventBus
	apiURL                  string
	apiToken                string
}

// NewSandbox creates a new dockerFunctionInstance of the sandbox
// Note: It is essential to listen to the .Logs() event channel (probably in a for loop in go routine) as soon as possible
// after instantiation.
func NewSandbox(config *config.Config, apiURL string, apiToken string, eventBus eventbus.EventBus) (*Sandbox, error) {
	sb := &Sandbox{
		config:           config,
		runningFunctions: map[string]FunctionInstance{},
		runningJobs:      map[string]JobInstance{},
		eventBus:         eventBus,
		done:             make(chan struct{}),
		apiURL:           apiURL,
		apiToken:         apiToken,
	}
	if config.SandboxCleanupInterval != 0 {
		sb.ticker = time.NewTicker(config.SandboxCleanupInterval)
		go sb.cleanupJob()
	}
	if !config.UseSystemDeno {
		if err := ensureDeno(config); err != nil {
			return nil, errors.Wrap(err, "ensure deno")
		}
	}
	return sb, nil
}

// Function looks up a running function dockerFunctionInstance, or boots up an instance if it doesn't have one yet
// It also performs initialization (cals the init()) function, errors out when no running server runs in time
func (s *Sandbox) Function(ctx context.Context, name string, functionConfig *definition.FunctionConfig, code string) (FunctionInstance, error) {
	var (
		inst FunctionInstance
		err  error
		ok   bool
	)
	s.internalStateUpdateLock.RLock()
	inst, ok = s.runningFunctions[name]
	s.internalStateUpdateLock.RUnlock()

	if !ok {
		if functionConfig.Runtime == "" {
			functionConfig.Runtime = DefaultRuntime
		}

		builder, ok := runtimeFunctionInstantiators[functionConfig.Runtime]
		if !ok {
			return nil, fmt.Errorf("Unsupported runtime: %s", functionConfig.Runtime)
		}

		inst, err = builder(ctx, s.config, s.apiURL, s.apiToken, RunModeFunction, name, s.eventBus, functionConfig, code)

		if err != nil {
			return nil, err
		}
		s.internalStateUpdateLock.Lock()
		s.runningFunctions[name] = inst
		s.internalStateUpdateLock.Unlock()
	}
	return inst, nil
}

func (s *Sandbox) Job(ctx context.Context, name string, functionConfig *definition.FunctionConfig, code string) (JobInstance, error) {
	var (
		inst JobInstance
		err  error
		ok   bool
	)

	s.internalStateUpdateLock.RLock()
	inst, ok = s.runningJobs[name]
	s.internalStateUpdateLock.RUnlock()
	if !ok {
		if functionConfig.Runtime == "" {
			functionConfig.Runtime = DefaultRuntime
		}

		builder, ok := runtimeJobInstantiators[functionConfig.Runtime]
		if !ok {
			return nil, fmt.Errorf("Unsupported runtime: %s", functionConfig.Runtime)
		}

		inst, err = builder(ctx, s.config, s.apiURL, s.apiToken, name, s.eventBus, functionConfig, code)
		if err != nil {
			return nil, err
		}

		s.internalStateUpdateLock.Lock()
		s.runningJobs[name] = inst
		s.internalStateUpdateLock.Unlock()
	}
	return inst, nil
}

func (s *Sandbox) cleanup() {
	now := time.Now()
	if len(s.runningFunctions) == 0 {
		return
	}
	log.Debugf("Cleaning up %d running functions...", len(s.runningFunctions))
	for id, inst := range s.runningFunctions {
		if inst.LastInvoked().Add(s.config.SandboxFunctionKeepAlive).Before(now) {
			log.Debugf("Killing function '%s'.", inst.Name())
			if err := inst.Kill(); err != nil {
				log.Errorf("Error killing function instance: %s", err)
			}
			delete(s.runningFunctions, id)
		}
	}
}

func (s *Sandbox) Close() {
	// Close the cleanup ticker
	if s.ticker != nil {
		s.ticker.Stop()
		close(s.done)
	}
	s.Flush()
}

func (s *Sandbox) cleanupJob() {
	for {
		select {
		case <-s.done:
			return
		case <-s.ticker.C:
			s.cleanup()
		}
	}
}

func (s *Sandbox) Eject(funcName string) error {
	inst, ok := s.runningFunctions[funcName]
	if !ok {
		return fmt.Errorf("No such function: %s", funcName)
	}
	if err := inst.Kill(); err != nil {
		return err
	}
	delete(s.runningFunctions, funcName)
	return nil
}

func (s *Sandbox) Flush() {
	// Kill all running function instances
	log.Infof("Stopping %d running functions...", len(s.runningFunctions))
	for fnName := range s.runningFunctions {
		if err := s.Eject(fnName); err != nil {
			log.Errorf("Could not stop %s: %s", fnName, err)
		}
	}
	s.runningFunctions = map[string]FunctionInstance{}

	// Stop all running jobs
	log.Infof("Stopping %d running jobs...", len(s.runningJobs))
	for _, inst := range s.runningJobs {
		if err := inst.Stop(); err != nil {
			log.Errorf("Error stopping job %s: %s", inst.Name(), err)
		}
	}
	s.runningJobs = map[string]JobInstance{}
}
