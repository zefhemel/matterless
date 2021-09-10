package sandbox

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
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
		done:           make(chan struct{}, 1),
	}

	if err := ew.start(); err != nil {
		return nil, errors.Wrap(err, "job start")
	}

	return ew, err
}

func (ew *JobExecutionWorker) log(funcName, message string) {
	ew.ceb.Publish(fmt.Sprintf("function.%s.log", ew.name), []byte(message))
}

func (ew *JobExecutionWorker) start() error {
	var (
		err error
	)
	// TODO: Do something better here?
	ctx := context.Background()

	// One invoke at a time per worker
	ew.functionExecutionLock.Lock()
	defer ew.functionExecutionLock.Unlock()

	if ew.runningInstance != nil {
		return errors.New("job already running")
	}

	if ew.functionConfig.Runtime == "" {
		ew.functionConfig.Runtime = DefaultRuntime
	}

	builder, ok := runtimeJobInstantiators[ew.functionConfig.Runtime]
	if !ok {
		return fmt.Errorf("unsupported runtime: %s", ew.functionConfig.Runtime)
	}

	ew.runningInstance, err = builder(ctx, ew.config, ew.apiURL, ew.apiToken, ew.name, ew.log, ew.functionConfig, ew.code)

	if err != nil {
		return err
	}

	go func() {
		select {
		case <-ew.done:
			return
		case <-ew.runningInstance.DidExit():
			log.Infof("Job process exited")
			ew.runningInstance = nil
		}
	}()

	return ew.runningInstance.Start(ctx)
}

func (ew *JobExecutionWorker) Close() error {
	close(ew.done)
	// Stop running instance if any
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ew.runningInstance.Stop(ctx); err != nil {
		return errors.Wrap(err, "closing worker")
	}
	return nil
}
