package sandbox_test

import (
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

func TestDenoSandboxFunction(t *testing.T) {
	cfg := config.NewConfig()
	cfg.DataDir = os.TempDir()
	cfg.UseSystemDeno = true

	sillyEvent := map[string]string{
		"name": "Zef",
	}
	code := `
	function handle(evt) {
		console.log('Log message');
		return {
			status: "ok"
		};
	}
	`

	// Boot up cluster
	conn, err := cluster.ConnectOrBoot("nats://localhost:4222")
	assert.NoError(t, err)
	ceb := cluster.NewClusterEventBus(conn, "test")

	// Listen to logs
	ceb.Subscribe("function.*.log", func(msg *nats.Msg) {
		log.Infof("Got log: %s", msg.Data)
	})

	// Boot worker
	worker, err := sandbox.NewFunctionExecutionWorker(cfg, "http://%s", "", ceb, "TestFunction", &definition.FunctionConfig{
		Runtime: "deno",
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

func TestDenoSandboxJob(t *testing.T) {
	cfg := config.NewConfig()
	cfg.DataDir = os.TempDir()
	cfg.UseSystemDeno = true

	code := `
function init(config) {
	console.log("Got config", config, "and env", Deno.env.get("API_URL"));
}

function start() {
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
	worker, err := sandbox.NewJobExecutionWorker(cfg, "http://%s", "", ceb, "TestJob", &definition.FunctionConfig{
		Runtime: "deno",
		Init: map[string]interface{}{
			"something": "To do",
		},
	}, code)
	assert.NoError(t, err)

	// Start Job
	_, err = ceb.InvokeFunction("TestJob", struct{}{})
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)
	assert.NoError(t, worker.Close())
	assert.Contains(t, allLogs, "http://localhost")
	assert.Contains(t, allLogs, "To do")
	assert.Contains(t, allLogs, "Iteration")
	assert.Contains(t, allLogs, "Stopping")
	// t.Fail()
}
