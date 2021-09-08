package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
)

type JobExecutionWorker struct {
	apiURL   string
	apiToken string
	config   *config.Config
	done     chan struct{}

	ceb *cluster.ClusterEventBus

	name           string
	functionConfig *definition.FunctionConfig
	code           string
	subscription   cluster.Subscription

	functionExecutionLock sync.Mutex
	runningInstance       JobInstance
}

func NewJobExecutionWorker(
	cfg *config.Config, apiURL string, apiToken string, ceb *cluster.ClusterEventBus,
	name string, functionConfig *definition.FunctionConfig, code string) (*JobExecutionWorker, error) {

	var err error
	ew := &JobExecutionWorker{
		config:         cfg,
		ceb:            ceb,
		apiURL:         apiURL,
		apiToken:       apiToken,
		name:           name,
		functionConfig: functionConfig,
		code:           code,
		done:           make(chan struct{}),
	}

	if ew.subscription, err = ceb.SubscribeInvokeFunction(name, func(i interface{}) (interface{}, error) {
		return nil, ew.start()
	}); err != nil {
		return nil, err
	}

	return ew, err
}

func (ew *JobExecutionWorker) log(funcName, message string) {
	ew.ceb.Publish(fmt.Sprintf("function.%s.log", ew.name), []byte(message))
}

func (ew *JobExecutionWorker) start() error {
	var (
		inst JobInstance
		err  error
	)
	// TODO: Do something better here?
	ctx := context.Background()

	// One invoke at a time per worker
	ew.functionExecutionLock.Lock()
	defer ew.functionExecutionLock.Unlock()
	inst = ew.runningInstance

	if inst == nil {
		if ew.functionConfig.Runtime == "" {
			ew.functionConfig.Runtime = DefaultRuntime
		}

		builder, ok := runtimeJobInstantiators[ew.functionConfig.Runtime]
		if !ok {
			return fmt.Errorf("unsupported runtime: %s", ew.functionConfig.Runtime)
		}

		inst, err = builder(ctx, ew.config, ew.apiURL, ew.apiToken, ew.name, ew.log, ew.functionConfig, ew.code)

		if err != nil {
			return err
		}
		ew.runningInstance = inst
	}
	return inst.Start(ctx)
}

func (ew *JobExecutionWorker) Close() error {
	// Stop running instance if any
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if ew.runningInstance != nil {
		if err := ew.runningInstance.Stop(ctx); err != nil {
			return errors.Wrap(err, "closing worker")
		}
	}

	// Unsubscribe from queue
	return ew.subscription.Unsubscribe()
}
