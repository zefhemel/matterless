package sandbox_test

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

func TestDenoSandboxFunction(t *testing.T) {
	sillyEvent := map[string]string{
		"name": "Zef",
	}
	eventBus := eventbus.NewLocalEventBus()
	s := sandbox.NewSandbox("", eventBus, 10*time.Second, 15*time.Second)
	eventBus.Subscribe("logs:*", func(eventName string, eventData interface{}) {
		logEntry := eventData.(sandbox.LogEntry)
		log.Infof("Got log: %s", logEntry.Message)
	})
	defer s.Close()
	code := `
	function handle(evt) {
		console.log('Log message');
		return {
			status: "ok:" + Deno.env.get("ENVVAR")
		};
	}
	`
	env := sandbox.EnvMap(map[string]string{
		"ENVVAR": "VALUE",
	})
	modules := sandbox.ModuleMap(map[string]string{})
	functionConfig := definition.FunctionConfig{
		Runtime: "deno",
	}

	// Init
	funcInstance, err := s.Function(context.Background(), "test", env, modules, functionConfig, code)
	assert.NoError(t, err)

	// Invoke
	for i := 0; i < 10; i++ {
		result, err := funcInstance.Invoke(context.Background(), sillyEvent)
		assert.NoError(t, err)
		assert.Equal(t, "ok:VALUE", result.(map[string]interface{})["status"])
	}
}

func TestDenoSandboxJob(t *testing.T) {
	eventBus := eventbus.NewLocalEventBus()
	s := sandbox.NewSandbox("", eventBus, 10*time.Second, 15*time.Second)
	logCounter := 0
	eventBus.Subscribe("logs:*", func(eventName string, eventData interface{}) {
		logEntry := eventData.(sandbox.LogEntry)
		log.Infof("Got log: %s", logEntry.Message)
		logCounter++
	})
	defer s.Close()
	code := `
	function init(config) {
       console.log("Got config", config, "and env", Deno.env.get("ENVVAR"));
   }

	function start() {
	}

   function run() {
       console.log("Running");
		setInterval(() => {
           console.log("Iteration");
       }, 500);
   }

   function stop() {
       console.log("Stopping");
   }
	`
	env := sandbox.EnvMap(map[string]string{
		"ENVVAR": "VALUE",
	})
	modules := sandbox.ModuleMap(map[string]string{})

	// Init
	jobInstance, err := s.Job(context.Background(), "test", env, modules, definition.FunctionConfig{
		Config: map[string]interface{}{
			"something": "To do",
		},
		Runtime: "deno",
	}, code)
	assert.NoError(t, err)

	err = jobInstance.Start(context.Background())
	assert.NoError(t, err)
	time.Sleep(2 * time.Second)
	// Some iteration logs should have been written
	assert.True(t, logCounter > 5)
}
