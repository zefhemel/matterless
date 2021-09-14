package sandbox

import (
	"context"
	"fmt"
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

	if !functionConfig.Prewarm && cfg.SandboxCleanupInterval != 0 {
		// Cleanup job
		fm.ticker = time.NewTicker(cfg.SandboxCleanupInterval)
		go fm.cleanupJob()
	}

	if functionConfig.Prewarm {
		if err := fm.warmup(context.Background()); err != nil {
			return nil, err
		}
	}

	return fm, err
}

func (fm *FunctionExecutionWorker) log(funcName, message string) {
	fm.ceb.PublishLog(fm.name, message)
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

func (fm *FunctionExecutionWorker) invoke(event interface{}) (interface{}, error) {
	// One invoke at a time per worker
	fm.functionExecutionLock.Lock()
	defer fm.functionExecutionLock.Unlock()
	// TODO: Do something better here?
	ctx := context.Background()

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

func (fm *FunctionExecutionWorker) Close() error {
	// Close the cleanup ticker
	if fm.ticker != nil {
		fm.ticker.Stop()
		close(fm.done)
	}

	// Stop running instance if any
	if fm.runningInstance != nil {
		fm.runningInstance.Kill()
		fm.runningInstance = nil
	}

	// Unsubscribe from queue
	return fm.subscription.Unsubscribe()
}
