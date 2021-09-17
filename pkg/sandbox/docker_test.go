package sandbox_test

import (
	"os"
	"testing"
	"time"

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

	conn, err := cluster.ConnectOrBoot(cfg)
	assert.NoError(t, err)
	ceb := cluster.NewClusterEventBus(conn, "test")

	// Listen to logs
	ceb.SubscribeLogs("*", func(funcName, message string) {
		log.Infof("Got log: %s", funcName)
	})

	// Boot worker
	worker, err := sandbox.NewFunctionExecutionWorker(cfg, "", "", ceb, "TestFunction", &definition.FunctionConfig{
		Runtime:     "docker",
		DockerImage: "zefhemel/mls-node-function",
	}, code, definition.LibraryMap{})
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
import os
import time

print("API_URL", os.getenv("API_URL"))
for i in range(10):
    print("Iteration")
    time.sleep(0.2)
`

	// Boot up cluster
	conn, err := cluster.ConnectOrBoot(cfg)
	assert.NoError(t, err)
	ceb := cluster.NewClusterEventBus(conn, "test2")

	// Listen to logs
	allLogs := ""
	ceb.SubscribeLogs("*", func(funcName, message string) {
		log.Infof("Got log: %s", message)
		allLogs = allLogs + string(message)
	})

	// Boot worker
	worker, err := sandbox.NewJobExecutionWorker(cfg, "http://%s", "", ceb, "TestJob", &definition.JobConfig{
		Runtime:     "docker",
		DockerImage: "zefhemel/mls-python3-job",
	}, code, definition.LibraryMap{})
	assert.NoError(t, err)

	// Start Job
	time.Sleep(1 * time.Second)
	assert.NoError(t, worker.Close())
	assert.Contains(t, allLogs, "http://host.docker.internal")
	//assert.Contains(t, allLogs, "To do")
	assert.Contains(t, allLogs, "Iteration")
	//assert.Contains(t, allLogs, "Stopping")
	// t.Fail()
}
