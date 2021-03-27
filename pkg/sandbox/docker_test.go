package sandbox_test

import (
	"context"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

func TestDockerSandboxFunction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	sillyEvent := map[string]string{
		"name": "Zef",
	}
	s := sandbox.NewDockerSandbox(10*time.Second, 15*time.Second)
	s.EventBus().Subscribe("logs:*", func(eventName string, eventData interface{}) (interface{}, error) {
		logEntry := eventData.(sandbox.LogEntry)
		log.Infof("Got log: %s", logEntry.Message)
		return nil, nil
	})
	defer s.Close()
	code := `
	function handle(evt) {
		console.log('Log message');
		if(evt.name === "Zef") {
			return {
				status: "ok:" + process.env.ENVVAR
			};
		} else {
			return {
				status: "error"
			};
		}
	}
	`
	env := sandbox.EnvMap(map[string]string{
		"ENVVAR": "VALUE",
	})
	modules := sandbox.ModuleMap(map[string]string{})

	// Init
	funcInstance, err := s.Function(context.Background(), "test", env, modules, code)
	assert.NoError(t, err)

	// Invoke
	for i := 0; i < 10; i++ {
		result, err := funcInstance.Invoke(context.Background(), sillyEvent)
		assert.NoError(t, err)
		assert.Equal(t, "ok:VALUE", result.(map[string]interface{})["status"])
	}
}

func TestDockerSandboxJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	s := sandbox.NewDockerSandbox(10*time.Second, 15*time.Second)
	logCounter := 0
	s.EventBus().Subscribe("logs:*", func(eventName string, eventData interface{}) (interface{}, error) {
		logEntry := eventData.(sandbox.LogEntry)
		log.Infof("Got log: %s", logEntry.Message)
		logCounter++
		return nil, nil
	})
	defer s.Close()
	code := `
	function start(params) {
		console.log("Params", params);
        return {
           MY_TOKEN: "1234"
        };
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
	jobInstance, err := s.Job(context.Background(), "test", env, modules, code)
	assert.NoError(t, err)

	envM, err := jobInstance.Start(context.Background(), map[string]interface{}{
		"something": "To do",
	})
	assert.NoError(t, err)
	time.Sleep(2 * time.Second)
	log.Info("Env", envM, logCounter)
	// Some iteration logs should have been written
	assert.True(t, logCounter > 5)
}
