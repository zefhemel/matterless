package sandbox

import (
	"context"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"sync"
	"time"
)

type LogEntry struct {
	Instance FunctionInstance
	Message  string
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
	Stop()
	Kill() error
}

type Sandbox struct {
	runningFunctions map[string]FunctionInstance
	runningJobs      map[string]JobInstance // key: job name
	cleanupInterval  time.Duration
	keepAlive        time.Duration
	ticker           *time.Ticker
	stop             chan struct{}
	bootLock         sync.Mutex
	eventBus         eventbus.EventBus
	config           *config.Config
	apiURL           string
	apiToken         string
}

// NewSandbox creates a new dockerFunctionInstance of the sandbox
// Note: It is essential to listen to the .Logs() event channel (probably in a for loop in go routine) as soon as possible
// after instantiation.
func NewSandbox(config *config.Config, apiURL string, apiToken string, eventBus eventbus.EventBus, cleanupInterval time.Duration, keepAlive time.Duration) (*Sandbox, error) {
	sb := &Sandbox{
		cleanupInterval:  cleanupInterval,
		keepAlive:        keepAlive,
		runningFunctions: map[string]FunctionInstance{},
		runningJobs:      map[string]JobInstance{},
		eventBus:         eventBus,
		stop:             make(chan struct{}),
		apiURL:           apiURL,
		apiToken:         apiToken,
		config:           config,
	}
	if cleanupInterval != 0 {
		sb.ticker = time.NewTicker(cleanupInterval)
		go sb.cleanupJob()
	}
	// TODO: Reenable auto download
	//if err := ensureDeno(config); err != nil {
	//	return nil, errors.Wrap(err, "ensure deno")
	//}
	return sb, nil
}

// Function looks up a running function dockerFunctionInstance, or boots up an instance if it doesn't have one yet
// It also performs initialization (cals the init()) function, errors out when no running server runs in time
func (s *Sandbox) Function(ctx context.Context, name string, functionConfig definition.FunctionConfig, code string) (FunctionInstance, error) {
	// Only one function can be booted at once for now
	// TODO: Remove this restriction
	s.bootLock.Lock()
	defer s.bootLock.Unlock()

	var (
		inst FunctionInstance
		err  error
		ok   bool
	)
	inst, ok = s.runningFunctions[name]
	if !ok {
		if functionConfig.Runtime == "" {
			functionConfig.Runtime = DefaultRuntime
		}
		switch functionConfig.Runtime {
		case "node":
			inst, err = newDockerFunctionInstance(ctx, s.config, s.apiURL, s.apiToken, "node-function", name, s.eventBus, functionConfig, code)
		case "deno":
			inst, err = newDenoFunctionInstance(ctx, s.config, s.apiURL, s.apiToken, "function", name, s.eventBus, functionConfig, code)
		}
		if inst == nil {
			return nil, errors.New("invalid runtime")
		}
		if err != nil {
			return nil, err
		}
		s.runningFunctions[name] = inst
	}
	return inst, nil
}

func (s *Sandbox) Job(ctx context.Context, name string, functionConfig definition.FunctionConfig, code string) (JobInstance, error) {
	// Only one function can be booted at once for now
	// TODO: Remove this restriction
	s.bootLock.Lock()
	defer s.bootLock.Unlock()

	var (
		inst JobInstance
		err  error
		ok   bool
	)

	inst, ok = s.runningJobs[name]
	if !ok {
		if functionConfig.Runtime == "" {
			functionConfig.Runtime = DefaultRuntime
		}
		switch functionConfig.Runtime {
		case "node":
			inst, err = newDockerJobInstance(ctx, s.config, s.apiURL, s.apiToken, name, s.eventBus, functionConfig, code)
		case "deno":
			inst, err = newDenoJobInstance(ctx, s.config, s.apiURL, s.apiToken, name, s.eventBus, functionConfig, code)
		}
		if inst == nil {
			return nil, errors.New("invalid runtime")
		}
		if err != nil {
			return nil, err
		}
		s.runningJobs[name] = inst
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
		if inst.LastInvoked().Add(s.keepAlive).Before(now) {
			log.Debugf("Killing dockerFunctionInstance '%s'.", inst.Name())
			if err := inst.Kill(); err != nil {
				log.Error("Error killing dockerFunctionInstance", err)
			}
			// Is it ok to delete entries from a map while iterating over it?
			delete(s.runningFunctions, id)
		}
	}
}

func (s *Sandbox) Close() {
	// Close the cleanup ticker
	if s.ticker != nil {
		s.ticker.Stop()
		s.stop <- struct{}{}
	}
	s.Flush()
}

func (s *Sandbox) cleanupJob() {
	for {
		select {
		case <-s.stop:
			return
		case <-s.ticker.C:
			s.cleanup()
		}
	}
}

func (s *Sandbox) Flush() {
	// Close all running instances
	log.Infof("Stopping %d running functions...", len(s.runningFunctions))
	for _, inst := range s.runningFunctions {
		if err := inst.Kill(); err != nil {
			log.Error("Error killing dockerFunctionInstance", err)
		}
	}
	s.runningFunctions = map[string]FunctionInstance{}
	// Close all running jobs
	log.Infof("Stopping %d running jobs...", len(s.runningJobs))
	for _, inst := range s.runningJobs {
		if err := inst.Kill(); err != nil {
			log.Error("Error killing dockerJobInstance", err)
		}
	}
	s.runningJobs = map[string]JobInstance{}
}
