package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
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
}

func NewFunctionExecutionWorker(
	cfg *config.Config, apiURL string, apiToken string, ceb *cluster.ClusterEventBus,
	name string, functionConfig *definition.FunctionConfig, code string) (*FunctionExecutionWorker, error) {

	var err error
	fm := &FunctionExecutionWorker{
		config:         cfg,
		ceb:            ceb,
		apiURL:         apiURL,
		apiToken:       apiToken,
		name:           name,
		functionConfig: functionConfig,
		code:           code,
		done:           make(chan struct{}),
	}

	fm.subscription, err = ceb.QueueSubscribe(fmt.Sprintf("function.%s", name), fmt.Sprintf("function.%s.workers", fm.name), func(msg *nats.Msg) {
		var requestMessage cluster.FunctionInvoke
		if err := json.Unmarshal(msg.Data, &requestMessage); err != nil {
			log.Errorf("Could not unmarshal event data: %s", err)
			err = msg.Respond([]byte(util.MustJsonByteSlice(cluster.FunctionResult{
				IsError: true,
				Error:   err.Error(),
			})))
			if err != nil {
				log.Errorf("Could not respond with error message: %s", err)
			}
			return
		}
		resp, err := fm.invoke(requestMessage.Data)
		if err != nil {
			log.Errorf("Error executing function: %s", err)
			err = msg.Respond([]byte(util.MustJsonByteSlice(cluster.FunctionResult{
				IsError: true,
				Error:   err.Error(),
			})))
			if err != nil {
				log.Errorf("Could not respond with error message: %s", err)
			}
			return
		}
		err = msg.Respond([]byte(util.MustJsonByteSlice(cluster.FunctionResult{
			Data: resp,
		})))
		if err != nil {
			log.Errorf("Could not respond with response: %s", err)
		}
	})
	if err != nil {
		return nil, err
	}

	if cfg.SandboxCleanupInterval != 0 {
		fm.ticker = time.NewTicker(cfg.SandboxCleanupInterval)
		go fm.cleanupJob()
	}

	return fm, err
}

func (fm *FunctionExecutionWorker) log(funcName, message string) {
	fm.ceb.Publish(fmt.Sprintf("function.%s.log", fm.name), []byte(message))
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
		if err := fm.runningInstance.Kill(); err != nil {
			log.Errorf("Error killing function instance: %s", err)
		}
		fm.runningInstance = nil
	}
}

func (fm *FunctionExecutionWorker) invoke(event interface{}) (interface{}, error) {
	var (
		inst FunctionInstance
		err  error
	)
	// TODO: Do something better here?
	ctx := context.Background()
	// TODO: Don't limit to one function instantiation at a time
	fm.functionExecutionLock.Lock()
	defer fm.functionExecutionLock.Unlock()
	inst = fm.runningInstance

	if inst == nil {
		if fm.functionConfig.Runtime == "" {
			fm.functionConfig.Runtime = DefaultRuntime
		}

		builder, ok := runtimeFunctionInstantiators[fm.functionConfig.Runtime]
		if !ok {
			return nil, fmt.Errorf("unsupported runtime: %s", fm.functionConfig.Runtime)
		}

		inst, err = builder(ctx, fm.config, fm.apiURL, fm.apiToken, RunModeFunction, fm.name, fm.log, fm.functionConfig, fm.code)

		if err != nil {
			return nil, err
		}
		fm.runningInstance = inst
	}
	return inst.Invoke(ctx, event)
}

func (fm *FunctionExecutionWorker) Close() error {
	// Close the cleanup ticker
	if fm.ticker != nil {
		fm.ticker.Stop()
		close(fm.done)
	}

	// Stop running instance if any
	if fm.runningInstance != nil {
		if err := fm.runningInstance.Kill(); err != nil {
			return errors.Wrap(err, "closing worker")
		}
	}

	// Unsubscribe from queue
	return fm.subscription.Unsubscribe()
}
