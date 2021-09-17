package application

import (
	"encoding/json"
	"fmt"
	"github.com/zefhemel/matterless/pkg/definition"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/store"
	"github.com/zefhemel/matterless/pkg/util"
)

type Container struct {
	config                *config.Config
	clusterConn           *nats.Conn
	clusterEventBus       *cluster.ClusterEventBus
	clusterLeaderElection *cluster.LeaderElection
	clusterStore          *store.JetstreamStore
	apps                  map[string]*Application
	apiGateway            *APIGateway
	done                  chan struct{}
	desiredStateLock      sync.Mutex
}

const (
	AdminTokenKey = "AdminToken"
)

func NewContainer(config *config.Config) (*Container, error) {
	var err error
	appMap := map[string]*Application{}
	c := &Container{
		config: config,
		apps:   appMap,
	}

	if err = os.MkdirAll(config.DataDir, 0700); err != nil {
		return nil, errors.Wrap(err, "create data dir")
	}

	c.clusterConn, err = cluster.ConnectOrBoot(config)
	if err != nil {
		return nil, errors.Wrap(err, "create container nats")
	}

	c.clusterLeaderElection, err = cluster.NewLeaderElection(c.clusterConn, fmt.Sprintf("%s.currentLeader", config.ClusterNatsPrefix), fmt.Sprintf("%s.heartbeat", config.ClusterNatsPrefix), 2*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "leader election")
	}

	c.clusterEventBus = cluster.NewClusterEventBus(c.clusterConn, config.ClusterNatsPrefix)

	clusterWrappedStore, err := store.NewLevelDBStore(fmt.Sprintf("%s/.cluster_store", config.DataDir))
	if err != nil {
		return nil, errors.Wrap(err, "create cluster leveldb store")
	}

	c.clusterStore, err = store.NewJetstreamStore(c.clusterConn, fmt.Sprintf("%s_cluster", config.ClusterNatsPrefix), clusterWrappedStore)
	if err != nil {
		return nil, errors.Wrap(err, "create cluster store")
	}
	if err := c.clusterStore.Connect(config.DatastoreSyncTimeout); err != nil {
		return nil, errors.Wrap(err, "cluster store connect")
	}

	if config.AdminToken == "" {
		// Lookup token in cluster store
		adminToken, err := c.clusterStore.Get(AdminTokenKey)
		if err != nil {
			return nil, errors.Wrap(err, "admin token lookup")
		}

		// TODO: Race condition with multiple simultaneous connecting clients
		if adminToken == nil {
			config.AdminToken = util.TokenGenerator()
			if err := c.clusterStore.Put(AdminTokenKey, config.AdminToken); err != nil {
				return nil, errors.Wrap(err, "store generated admin token")
			}
		} else {
			config.AdminToken = adminToken.(string)
		}
	}

	c.subscribeToEvents()

	return c, nil
}

func (c *Container) Start() error {
	c.apiGateway = NewAPIGateway(c.config, c)
	if err := c.apiGateway.Start(); err != nil {
		return err
	}

	if c.config.LoadApps {
		if err := c.loadApps(); err != nil {
			return err
		}
	}
	c.done = make(chan struct{})
	go c.monitorCluster()

	return nil
}

func (c *Container) subscribeToEvents() {

	c.clusterStore.SubscribePuts(func(event store.PutMessage) {
		if strings.HasPrefix(event.Key, "app:") {
			var err error
			appName := event.Key[len("app:"):]
			app := c.Get(appName)

			log.Infof("Loading app %s...", appName)
			if app == nil {
				app, err = c.CreateApp(appName)
				if err != nil {
					log.Errorf("Could not create app: %s", appName)
					return
				}
			}

			var defs definition.Definitions
			if err := json.Unmarshal(util.MustJsonByteSlice(event.Value), &defs); err != nil {
				log.Errorf("Could not unmarshall definitions: %s", err)
				return
			}

			if err := app.Eval(&defs); err != nil {
				log.Errorf("Could not evaluate app: %s", err)
				return
			}

			if c.clusterLeaderElection.IsLeader() {
				if err := c.bringToDesiredState(); err != nil {
					log.Errorf("Could not bring cluster to desired state: %s", err)
				}
				if err := app.PublishAppEvent("init", struct{}{}); err != nil {
					log.Errorf("could not send init event: %s", err)
				}
			}
		}
	})

	c.clusterStore.SubscribeDeletes(func(event store.DeleteMessage) {
		if strings.HasPrefix(event.Key, "app:") {
			appName := event.Key[len("app:"):]

			log.Infof("Deleting app %s...", appName)
			if err := c.DeleteApp(appName); err != nil {
				log.Errorf("Could not delete app: %s", err)
			}
		}
	})

	c.clusterEventBus.SubscribeFetchClusterInfo(func() *cluster.NodeInfo {
		return c.NodeInfo()
	})

	c.clusterEventBus.SubscribeRestartApp(func(appName string) {
		app := c.Get(appName)
		if app == nil {
			log.Errorf("Asked to restart non-existing app: %s", appName)
			return
		}
		if err := app.Eval(app.unprocessedDefinitions); err != nil {
			log.Errorf("Error starting app: %s", err)
		}
		if c.clusterLeaderElection.IsLeader() {
			if err := c.bringToDesiredState(); err != nil {
				log.Errorf("Could not bring cluster to desired state: %s", err)
			}
			if err := app.PublishAppEvent("init", struct{}{}); err != nil {
				log.Errorf("could not send init event: %s", err)
			}
		}
	})
}

func (c *Container) loadApps() error {
	results, err := c.clusterStore.QueryPrefix("app:")
	if err != nil {
		return errors.Wrap(err, "query apps")
	}
	for _, result := range results {
		appName := result.Key[len("app:"):]
		var defs definition.Definitions
		if err := json.Unmarshal(util.MustJsonByteSlice(result.Value), &defs); err != nil {
			return errors.Wrap(err, "decode apps")
		}
		app, err := c.CreateApp(appName)
		if err != nil {
			return errors.Wrap(err, "create app")
		}
		if err := app.Eval(&defs); err != nil {
			return errors.Wrap(err, "eval app")
		}
	}
	if c.clusterLeaderElection.IsLeader() {
		if err := c.bringToDesiredState(); err != nil {
			return errors.Wrap(err, "desired state")
		}
	}

	return nil
}

func (c *Container) monitorCluster() {
	for {
		select {
		case <-c.done:
			return
		case <-time.After(10 * time.Second):
			if c.clusterLeaderElection.IsLeader() {
				c.bringToDesiredState()
			}
		}
	}
}

func (c *Container) bringToDesiredState() error {
	c.desiredStateLock.Lock()
	defer c.desiredStateLock.Unlock()
	clusterInfo, err := c.clusterEventBus.FetchClusterInfo(time.Second)
	if err != nil {
		return errors.Wrap(err, "fetch cluster info")
	}
	// Iterate over all apps
	for _, app := range c.apps {
		c.bringAppToDesiredState(app, clusterInfo)
	}

	return nil
}

func (c *Container) bringAppToDesiredState(app *Application, clusterInfo *cluster.ClusterInfo) {
	//functionInstancesToStart := map[definition.FunctionID]int{}
	jobInstancesToStart := map[definition.FunctionID]int{}

	// First collect all jobs desired instances
	for jobName, jobDef := range app.definitions.Jobs {
		jobInstancesToStart[jobName] = jobDef.Config.Instances
	}

	// Then iterate over all nodes and remove all running jobs
	for nodeId, nodeInfo := range clusterInfo.Nodes {
		appInfo, ok := nodeInfo.Apps[app.Name()]
		if !ok {
			log.Errorf("Node %d does not know about app %s", nodeId, app.Name())
			log.Errorf("All data: %+v", nodeInfo.Apps)
			continue
		}
		for jobName, runningInstances := range appInfo.JobWorkers {
			jobInstancesToStart[definition.FunctionID(jobName)] -= runningInstances
		}
	}

	// We're now left with a map with not running jobs, let's kick those off
	for jobName, toStart := range jobInstancesToStart {
		if toStart <= 0 {
			continue
		}
		log.Infof("Now requesting %d instances of %s", toStart, jobName)
		if err := app.eventBus.RequestJobWorkers(string(jobName), toStart, c.config.SandboxJobStartTimeout); err != nil {
			log.Errorf("Could not start workers: %s", err)
		}
	}
}

func (c *Container) CreateApp(appName string) (*Application, error) {
	appDataPath := fmt.Sprintf("%s/%s", c.config.DataDir, util.SafeFilename(appName))

	if err := os.MkdirAll(appDataPath, 0700); err != nil {
		return nil, errors.Wrap(err, "create data dir")
	}
	levelDBStore, err := store.NewLevelDBStore(appDataPath)
	if err != nil {
		return nil, errors.Wrap(err, "create data store dir")
	}

	jsStore, err := store.NewJetstreamStore(c.clusterConn, fmt.Sprintf("%s_%s", c.config.ClusterNatsPrefix, appName), levelDBStore)
	if err != nil {
		return nil, errors.Wrap(err, "create jetstream store")
	}

	if err := jsStore.Connect(c.config.DatastoreSyncTimeout); err != nil {
		return nil, errors.Wrap(err, "jetstream store connect")
	}

	app, err := NewApplication(c.config, appName, jsStore, cluster.NewClusterEventBus(c.clusterConn, fmt.Sprintf("%s.%s", c.config.ClusterNatsPrefix, appName)))
	if err != nil {
		return nil, err
	}

	c.apps[appName] = app

	return app, nil
}

func (c *Container) DeleteApp(name string) error {
	if app, ok := c.apps[name]; ok {
		if err := app.Close(); err != nil {
			return errors.Wrap(err, "closing app")
		}
		if err := app.dataStore.DeleteStore(); err != nil {
			return errors.Wrap(err, "delete store")
		}
		delete(c.apps, name)
		return nil
	} else {
		return errors.New("app not found")
	}
}

func (c *Container) Close() {
	close(c.done)
	for _, app := range c.apps {
		if err := app.Close(); err != nil {
			log.Errorf("Failed to cleanly shut down application %s: %s", app.appName, err)
		}
	}
	c.apiGateway.Stop()
}

func (c *Container) ClusterEventBus() *cluster.ClusterEventBus {
	return c.clusterEventBus
}

func (c *Container) Get(name string) *Application {
	return c.apps[name]
}

func (c *Container) Store() *store.JetstreamStore {
	return c.clusterStore
}

func (c *Container) List() []string {
	appNames := make([]string, 0, len(c.apps))
	for name := range c.apps {
		appNames = append(appNames, name)
	}
	return appNames
}

func (c *Container) NodeInfo() *cluster.NodeInfo {
	ni := &cluster.NodeInfo{
		ID:   c.clusterLeaderElection.ID,
		Apps: map[string]*cluster.AppInfo{},
	}
	for appName, app := range c.apps {
		ni.Apps[appName] = app.Sandbox().AppInfo()
	}
	return ni
}
