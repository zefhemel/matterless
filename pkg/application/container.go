package application

import (
	"fmt"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/store"
	"github.com/zefhemel/matterless/pkg/util"
)

type Container struct {
	config          *config.Config
	clusterConn     *nats.Conn
	clusterEventBus *cluster.ClusterEventBus
	clusterStore    *store.JetstreamStore
	apps            map[string]*Application
	apiGateway      *APIGateway
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

	c.clusterConn, err = cluster.ConnectOrBoot(config.NatsUrl)
	if err != nil {
		return nil, errors.Wrap(err, "create container nats")
	}

	c.clusterEventBus = cluster.NewClusterEventBus(c.clusterConn, config.NatsPrefix)

	clusterWrappedStore, err := store.NewLevelDBStore(fmt.Sprintf("%s/cluster_store", config.DataDir))
	if err != nil {
		return nil, errors.Wrap(err, "create cluster leveldb store")
	}

	c.clusterStore, err = store.NewJetstreamStore(c.clusterConn, fmt.Sprintf("%s_cluster", config.NatsPrefix), clusterWrappedStore)
	if err != nil {
		return nil, errors.Wrap(err, "create cluster store")
	}
	if err := c.clusterStore.Connect(); err != nil {
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

	c.apiGateway = NewAPIGateway(config, c)
	if err := c.apiGateway.Start(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Container) CreateApp(appName string) (*Application, error) {
	appDataPath := fmt.Sprintf("%s/%s", c.config.DataDir, util.SafeFilename(appName))
	// eventBus := eventbus.NewLocalEventBus()

	if err := os.MkdirAll(appDataPath, 0700); err != nil {
		return nil, errors.Wrap(err, "create data dir")
	}
	levelDBStore, err := store.NewLevelDBStore(fmt.Sprintf("%s/store", appDataPath))
	if err != nil {
		return nil, errors.Wrap(err, "create data store dir")
	}

	jsStore, err := store.NewJetstreamStore(c.clusterConn, fmt.Sprintf("%s_%s", c.config.NatsPrefix, appName), levelDBStore)
	if err != nil {
		return nil, errors.Wrap(err, "create jetstream store")
	}

	if err := jsStore.Connect(); err != nil {
		return nil, errors.Wrap(err, "jetstream store connect")
	}

	// TODO: Re-enabled evented store
	// dataStore := store.NewEventedStore(jsStore, eventBus)
	app, err := NewApplication(c.config, appName, jsStore, cluster.NewClusterEventBus(c.clusterConn, fmt.Sprintf("%s.%s", c.config.NatsPrefix, appName)))
	if err != nil {
		return nil, err
	}

	c.apps[appName] = app

	// Listen to sandbox log events, republish on cluster event bus
	// DISABLED because all going through cluster now
	// app.EventBus().Subscribe("logs:*", func(eventName string, eventData interface{}) {
	// 	if le, ok := eventData.(sandbox.LogEntry); ok {
	// 		if err := c.clusterConn.Publish(fmt.Sprintf("%s.logs.%s.%s", c.config.NatsPrefix, appName, le.FunctionName), util.MustJsonByteSlice(LogEntry{
	// 			AppName:  appName,
	// 			LogEntry: le,
	// 		})); err != nil {
	// 			log.Errorf("Could not publish log event: %s", err)
	// 		}
	// 	} else {
	// 		log.Fatal("Did not get sandbox.LogEntry", eventData)
	// 	}
	// })

	return app, nil
}

func (c *Container) Close() {
	for _, app := range c.apps {
		if err := app.Close(); err != nil {
			log.Errorf("Failed to cleanly shut down application %s: %s", app.appName, err)
		}
	}
	c.apiGateway.Stop()
}

func (c *Container) ClusterConnection() *nats.Conn {
	return c.clusterConn
}

func (c *Container) Get(name string) *Application {
	return c.apps[name]
}

func (c *Container) Deregister(name string) {
	if app, ok := c.apps[name]; ok {
		if err := app.Close(); err != nil {
			log.Errorf("Failed to cleanly stop app %s: %s", name, err)
		}
		delete(c.apps, name)
	}
}

// func (c *Container) LoadAppsFromDisk() error {
// 	files, err := os.ReadDir(c.config.DataDir)
// 	if err != nil {
// 		return err
// 	}
// fileLoop:
// 	for _, file := range files {
// 		appName := file.Name()
// 		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
// 			if _, err := os.Stat(fmt.Sprintf("%s/%s/application.md", c.config.DataDir, appName)); err != nil {
// 				// No application file
// 				continue fileLoop
// 			}
// 			// It's an app, let's load it
// 			app, err := NewApplication(c.config, appName)
// 			if err != nil {
// 				return errors.Wrapf(err, "creating app: %s", appName)
// 			}
// 			c.Register(appName, app)
// 			if err := app.LoadFromDisk(); err != nil {
// 				return errors.Wrapf(err, "loading app: %s", appName)
// 			}
// 		}
// 	}
// 	return nil
// }

func (c *Container) List() []string {
	appNames := make([]string, 0, len(c.apps))
	for name := range c.apps {
		appNames = append(appNames, name)
	}
	return appNames
}
