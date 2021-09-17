package sandbox_test

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"os"
	"testing"
	"time"
)

func TestDenoSandboxFunction(t *testing.T) {
	cfg := config.NewConfig()
	cfg.DataDir = os.TempDir()
	cfg.UseSystemDeno = true
	cfg.LoadApps = false

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
	conn, err := cluster.ConnectOrBoot(cfg)
	assert.NoError(t, err)
	ceb := cluster.NewClusterEventBus(conn, "test")

	// Listen to logs
	ceb.SubscribeLogs("*", func(funcName, message string) {
		log.Infof("Got log (func) %s", message)
	})

	// Boot worker
	worker, err := sandbox.NewFunctionExecutionWorker(cfg, "http://%s", "", ceb, "TestFunction", &definition.FunctionConfig{
		Runtime: "deno",
	}, code, map[definition.FunctionID]*definition.LibraryDef{})
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
	conn, err := cluster.ConnectOrBoot(cfg)
	assert.NoError(t, err)
	ceb := cluster.NewClusterEventBus(conn, "test2")

	// Listen to logs
	allLogs := ""
	ceb.SubscribeLogs("*", func(funcName, message string) {
		log.Infof("Got log (job) %s", message)

		allLogs = allLogs + string(message)
	})

	// Boot worker
	worker, err := sandbox.NewJobExecutionWorker(cfg, "http://%s", "", ceb, "TestJob", &definition.JobConfig{
		Runtime: "deno",
		Init: map[string]interface{}{
			"something": "To do",
		},
	}, code, definition.LibraryMap{})
	assert.NoError(t, err)

	// Start Job
	time.Sleep(1 * time.Second)
	assert.NoError(t, worker.Close())
	time.Sleep(200 * time.Millisecond)
	assert.Contains(t, allLogs, "http://localhost")
	assert.Contains(t, allLogs, "To do")
	assert.Contains(t, allLogs, "Iteration")
	assert.Contains(t, allLogs, "Stopping")
	// t.Fail()
}
