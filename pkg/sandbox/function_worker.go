package sandbox

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
)

type FunctionExecutionWorker struct {
	apiURL   string
	apiToken string
	config   *config.Config
	ticker   *time.Ticker
	done     chan struct{}

	ceb *cluster.ClusterEventBus

	name           string
	functionConfig *definition.FunctionConfig
	code           string
	subscription   cluster.Subscription

	functionExecutionLock sync.Mutex
	runningInstance       FunctionInstance
	invocationCount       int
	libs                  definition.LibraryMap
	cancelFn              context.CancelFunc
}

func NewFunctionExecutionWorker(
	cfg *config.Config, apiURL string, apiToken string, ceb *cluster.ClusterEventBus,
	name string, functionConfig *definition.FunctionConfig, code string, libs definition.LibraryMap) (*FunctionExecutionWorker, error) {

	var err error
	fm := &FunctionExecutionWorker{
		config:         cfg,
		ceb:            ceb,
		apiURL:         apiURL,
		apiToken:       apiToken,
		name:           name,
		functionConfig: functionConfig,
		libs:           libs,
		code:           code,
		done:           make(chan struct{}),
	}

	if fm.subscription, err = ceb.SubscribeInvokeFunction(name, fm.invoke); err != nil {
		return nil, err
	}

	everythingOK := false
	defer func() {
		if !everythingOK {
			log.Errorf("Could not properly start worker for %s, cleaning up", name)
			fm.Close()
		}
	}()

	if !functionConfig.Hot && cfg.SandboxCleanupInterval != 0 {
		// Cleanup job
		fm.ticker = time.NewTicker(cfg.SandboxCleanupInterval)
		go fm.cleanupJob()
	}

	if functionConfig.Hot {
		if err := fm.warmup(context.Background()); err != nil {
			return nil, err
		}
	}
	everythingOK = true
	return fm, err
}

func (fm *FunctionExecutionWorker) log(funcName, message string) {
	if err := fm.ceb.PublishLog(fm.name, message); err != nil {
		log.Errorf("Error publishing log: %s", err)
	}
}

func (fm *FunctionExecutionWorker) cleanupJob() {
	for {
		select {
		case <-fm.done:
			return
		case <-fm.ticker.C:
			fm.cleanup()
		}
	}
}

func (fm *FunctionExecutionWorker) cleanup() {
	if fm.runningInstance == nil {
		return
	}
	now := time.Now()
	if fm.runningInstance.LastInvoked().Add(fm.config.SandboxFunctionKeepAlive).Before(now) {
		log.Debugf("Killing function '%s'.", fm.runningInstance.Name())
		fm.runningInstance.Kill()
		fm.runningInstance = nil
	}
}

var FunctionStoppedErr = errors.New("function stopped")

func (fm *FunctionExecutionWorker) invoke(event interface{}) (interface{}, error) {
	// One invoke at a time per worker
	fm.functionExecutionLock.Lock()
	defer fm.functionExecutionLock.Unlock()
	var ctx context.Context
	ctx, fm.cancelFn = context.WithCancel(context.Background())

	if err := fm.warmup(ctx); err != nil {
		return nil, err
	}
	//log.Infof("Now actually locally invoking %s", fm.name)
	fm.invocationCount++
	return fm.runningInstance.Invoke(ctx, event)
}

func (fm *FunctionExecutionWorker) warmup(ctx context.Context) error {
	var err error
	inst := fm.runningInstance

	if inst == nil {
		if fm.functionConfig.Runtime == "" {
			fm.functionConfig.Runtime = DefaultRuntime
		}

		builder, ok := runtimeFunctionInstantiators[fm.functionConfig.Runtime]
		if !ok {
			return fmt.Errorf("unsupported runtime: %s", fm.functionConfig.Runtime)
		}

		inst, err = builder(ctx, fm.config, fm.apiURL, fm.apiToken, RunModeFunction, fm.name, fm.log, fm.functionConfig, fm.code, fm.libs)

		if err != nil {
			return err
		}
		fm.runningInstance = inst

		go func() {
			<-inst.DidExit()
			log.Info("Process exited, resetting running instance")
			fm.runningInstance = nil
		}()
	}
	return nil
}

func (fm *FunctionExecutionWorker) Close() {
	//log.Errorf("Closing worker %s", fm.name)
	if fm.cancelFn != nil {
		fm.cancelFn()
	}

	// Unsubscribe from queue
	if err := fm.subscription.Unsubscribe(); err != nil {
		log.Errorf("Could not unsubscribe function %s: %s", fm.name, err)
	}

	// Close the cleanup ticker
	if fm.ticker != nil {
		fm.ticker.Stop()
		close(fm.done)
	}

	// Stop running instance if any
	if fm.runningInstance != nil {
		fm.runningInstance.Kill()
		fm.runningInstance = nil
	} else {
		log.Errorf("No running instance for worker: %s", fm.name)
	}
}
