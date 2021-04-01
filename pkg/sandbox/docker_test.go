package sandbox_test

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

func TestDockerSandboxFunction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	cfg := &config.Config{
		DataDir: os.TempDir(),
	}

	sillyEvent := map[string]string{
		"name": "Zef",
	}
	eventBus := eventbus.NewLocalEventBus()
	s, err := sandbox.NewSandbox(cfg, "", "1234", eventBus, 10*time.Second, 15*time.Second)
	assert.NoError(t, err)
	eventBus.Subscribe("logs:*", func(eventName string, eventData interface{}) {
		logEntry := eventData.(sandbox.LogEntry)
		log.Infof("Got log: %s", logEntry.Message)
	})
	defer s.Close()
	code := `
	function handle(evt) {
		console.log('Log message');
		if(evt.name === "Zef") {
			return {
				status: "ok"
			};
		} else {
			return {
				status: "error"
			};
		}
	}
	`

	// Init
	funcInstance, err := s.Function(context.Background(), "test", definition.FunctionConfig{
		Runtime: "node",
	}, code)
	assert.NoError(t, err)

	// Invoke
	for i := 0; i < 10; i++ {
		result, err := funcInstance.Invoke(context.Background(), sillyEvent)
		assert.NoError(t, err)
		assert.Equal(t, "ok", result.(map[string]interface{})["status"])
	}
}

func TestDockerSandboxJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	cfg := &config.Config{
		DataDir: os.TempDir(),
	}
	eventBus := eventbus.NewLocalEventBus()
	s, err := sandbox.NewSandbox(cfg, "", "1234", eventBus, 10*time.Second, 15*time.Second)
	assert.NoError(t, err)
	logCounter := 0
	eventBus.Subscribe("logs:*", func(eventName string, eventData interface{}) {
		logEntry := eventData.(sandbox.LogEntry)
		log.Infof("Got log: %s", logEntry.Message)
		logCounter++
	})
	defer s.Close()
	code := `
	function init(config) {
        console.log("Got config", config);
    }

	function start() {
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

	// Init
	jobInstance, err := s.Job(context.Background(), "test", definition.FunctionConfig{
		Init: map[string]interface{}{
			"something": "To do",
		},
		Runtime: "node",
	}, code)
	assert.NoError(t, err)

	err = jobInstance.Start(context.Background())
	assert.NoError(t, err)
	time.Sleep(2 * time.Second)
	// Some iteration logs should have been written
	assert.True(t, logCounter > 5)
}
