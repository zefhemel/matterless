package sandbox

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"github.com/zefhemel/matterless/pkg/util"
	"sync"
	"time"
)

type LogEntry struct {
	Instance FunctionInstance
	Message  string
}

type ModuleMap map[string]string
type EnvMap map[string]string

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

type functionHash string

type Sandbox struct {
	runningFunctions map[functionHash]FunctionInstance
	runningJobs      map[string]JobInstance // key: job name
	cleanupInterval  time.Duration
	keepAlive        time.Duration
	ticker           *time.Ticker
	stop             chan struct{}
	bootLock         sync.Mutex
	eventBus         eventbus.EventBus
	config           *config.Config
}

// NewSandbox creates a new dockerFunctionInstance of the sandbox
// Note: It is essential to listen to the .Logs() event channel (probably in a for loop in go routine) as soon as possible
// after instantiation.
func NewSandbox(cfg *config.Config, eventBus eventbus.EventBus, cleanupInterval time.Duration, keepAlive time.Duration) *Sandbox {
	sb := &Sandbox{
		config:           cfg,
		cleanupInterval:  cleanupInterval,
		keepAlive:        keepAlive,
		runningFunctions: map[functionHash]FunctionInstance{},
		runningJobs:      map[string]JobInstance{},
		eventBus:         eventBus,
		stop:             make(chan struct{}),
	}
	if cleanupInterval != 0 {
		sb.ticker = time.NewTicker(cleanupInterval)
		go sb.cleanupJob()
	}
	return sb
}

// Function looks up a running function dockerFunctionInstance, or boots up an instance if it doesn't have one yet
// It also performs initialization (cals the init()) function, errors out when no running server runs in time
func (s *Sandbox) Function(ctx context.Context, name string, env EnvMap, modules ModuleMap, functionConfig definition.FunctionConfig, code string) (FunctionInstance, error) {
	// Only one function can be booted at once for now
	// TODO: Remove this restriction
	s.bootLock.Lock()
	defer s.bootLock.Unlock()

	functionHash := newFunctionHash(modules, env, functionConfig, code)
	var (
		inst FunctionInstance
		err  error
		ok   bool
	)
	inst, ok = s.runningFunctions[functionHash]
	if !ok {
		if functionConfig.Runtime == "" {
			functionConfig.Runtime = DefaultRuntime
		}
		switch functionConfig.Runtime {
		case "node":
			inst, err = newDockerFunctionInstance(ctx, s.config, "node-function", name, s.eventBus, env, modules, functionConfig, code)
		case "deno":
			inst, err = newDenoFunctionInstance(ctx, s.config, "function", name, s.eventBus, env, modules, functionConfig, code)
		}
		if inst == nil {
			return nil, errors.New("invalid runtime")
		}
		if err != nil {
			return nil, err
		}
		s.runningFunctions[functionHash] = inst
	}
	return inst, nil
}

func (s *Sandbox) Job(ctx context.Context, name string, env EnvMap, modules ModuleMap, functionConfig definition.FunctionConfig, code string) (JobInstance, error) {
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
			inst, err = newDockerJobInstance(ctx, s.config, name, s.eventBus, env, modules, functionConfig, code)
		case "deno":
			inst, err = newDenoJobInstance(ctx, s.config, name, s.eventBus, env, modules, functionConfig, code)
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
	s.runningFunctions = map[functionHash]FunctionInstance{}
	// Close all running jobs
	log.Infof("Stopping %d running jobs...", len(s.runningJobs))
	for _, inst := range s.runningJobs {
		if err := inst.Kill(); err != nil {
			log.Error("Error killing dockerJobInstance", err)
		}
	}
	s.runningJobs = map[string]JobInstance{}
}

func newFunctionHash(modules map[string]string, env map[string]string, functionConfig definition.FunctionConfig, code string) functionHash {
	// This can probably be optimized, the goal is to generate a unique string representing a mix of the code, modules and environment
	h := sha1.New()
	h.Write([]byte(util.MustJsonString(modules)))
	h.Write([]byte(util.MustJsonString(env)))
	h.Write([]byte(util.MustJsonString(functionConfig)))
	h.Write([]byte(code))
	bs := h.Sum(nil)
	return functionHash(fmt.Sprintf("%x", bs))
}
