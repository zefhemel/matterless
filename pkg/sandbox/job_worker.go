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

	name      string
	jobConfig *definition.JobConfig
	code      string

	functionExecutionLock sync.Mutex
	runningInstance       JobInstance
	libs                  definition.LibraryMap
}

func NewJobExecutionWorker(
	cfg *config.Config, apiURL string, apiToken string, ceb *cluster.ClusterEventBus,
	name string, jobConfig *definition.JobConfig, code string, libs definition.LibraryMap) (*JobExecutionWorker, error) {

	var err error

	ew := &JobExecutionWorker{
		config:    cfg,
		ceb:       ceb,
		apiURL:    apiURL,
		apiToken:  apiToken,
		name:      name,
		jobConfig: jobConfig,
		code:      code,
		libs:      libs,
		done:      make(chan struct{}, 1),
	}

	if err := ew.start(); err != nil {
		return nil, errors.Wrap(err, "job start")
	}

	return ew, err
}

func (ew *JobExecutionWorker) log(funcName, message string) {
	ew.ceb.PublishLog(ew.name, message)
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

	if ew.jobConfig.Runtime == "" {
		ew.jobConfig.Runtime = DefaultRuntime
	}

	builder, ok := runtimeJobInstantiators[ew.jobConfig.Runtime]
	if !ok {
		return fmt.Errorf("unsupported runtime: %s", ew.jobConfig.Runtime)
	}

	ew.runningInstance, err = builder(ctx, ew.config, ew.apiURL, ew.apiToken, ew.name, ew.log, ew.jobConfig, ew.code, ew.libs)

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
			close(ew.done)
		}
	}()

	return ew.runningInstance.Start(ctx)
}

func (ew *JobExecutionWorker) Close() error {
	close(ew.done)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ew.runningInstance.Stop(ctx); err != nil {
		return errors.Wrap(err, "closing worker")
	}
	return nil
}
