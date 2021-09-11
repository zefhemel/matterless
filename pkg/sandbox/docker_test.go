package sandbox_test

import (
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
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

	conn, err := cluster.ConnectOrBoot("nats://localhost:4222")
	assert.NoError(t, err)
	ceb := cluster.NewClusterEventBus(conn, "test")

	// Listen to logs
	ceb.Subscribe("function.*.log", func(msg *nats.Msg) {
		log.Infof("Got log: %s", msg.Data)
	})

	// Boot worker
	worker, err := sandbox.NewFunctionExecutionWorker(cfg, "", "", ceb, "TestFunction", &definition.FunctionConfig{
		Runtime: "node",
	}, code)
	assert.NoError(t, err)
	defer worker.Close()

	// Invoke
	for i := 0; i < 10; i++ {
		result, err := ceb.InvokeFunction("TestFunction", sillyEvent)
		assert.NoError(t, err)
		assert.Equal(t, "ok", result.(map[string]interface{})["status"])
	}
	// t.Fail()
}

func TestDockerSandboxJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := config.NewConfig()
	cfg.DataDir = os.TempDir()
	cfg.UseSystemDeno = true

	code := `
	function init(config) {
        console.log("Got config", config, " and env ", process.env);
    }

	function start() {
		console.log("Starting");
	}

    function run() {
        console.log("Running");
		setInterval(() => {
            console.log("Iteration");
        }, 50);
    }

    function stop() {
        console.log("Stopping");
    }
	`

	// Boot up cluster
	conn, err := cluster.ConnectOrBoot("nats://localhost:4222")
	assert.NoError(t, err)
	ceb := cluster.NewClusterEventBus(conn, "test")

	// Listen to logs
	allLogs := ""
	ceb.Subscribe("function.*.log", func(msg *nats.Msg) {
		log.Infof("Got log: %s", msg.Data)
		allLogs = allLogs + string(msg.Data)
	})

	// Boot worker
	worker, err := sandbox.NewJobExecutionWorker(cfg, "http://%s", "", ceb, "TestJob", &definition.JobConfig{
		Runtime: "node",
		Init: map[string]interface{}{
			"something": "To do",
		},
	}, code)
	assert.NoError(t, err)

	// Start Job
	time.Sleep(1 * time.Second)
	assert.NoError(t, worker.Close())
	assert.Contains(t, allLogs, "http://host.docker.internal")
	assert.Contains(t, allLogs, "To do")
	assert.Contains(t, allLogs, "Iteration")
	assert.Contains(t, allLogs, "Stopping")
	// t.Fail()
}
