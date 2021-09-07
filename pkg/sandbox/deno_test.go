package sandbox_test

import (
	"os"
	"testing"

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
	worker, err := sandbox.NewFunctionExecutionWorker(cfg, "", "", ceb, "TestFunction", &definition.FunctionConfig{
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

// func TestDenoSandboxJob(t *testing.T) {
// 	cfg := config.NewConfig()
// 	cfg.DataDir = os.TempDir()
// 	cfg.UseSystemDeno = true

// 	eventBus := eventbus.NewLocalEventBus()
// 	s, err := sandbox.NewSandbox(cfg, "", "1234", eventBus)
// 	assert.NoError(t, err)
// 	logCounter := 0
// 	eventBus.Subscribe("logs:*", func(eventName string, eventData interface{}) {
// 		logEntry := eventData.(sandbox.LogEntry)
// 		log.Infof("Got log: %s", logEntry.Message)
// 		logCounter++
// 	})
// 	defer s.Close()
// 	code := `
// 	function init(config) {
//        console.log("Got config", config, "and env", Deno.env.get("ENVVAR"));
//    }

// 	function start() {
// 	}

//    function run() {
//        console.log("Running");
// 		setInterval(() => {
//            console.log("Iteration");
//        }, 500);
//    }

//    function done() {
//        console.log("Stopping");
//    }
// 	`
// 	// Init
// 	jobInstance, err := s.Job(context.Background(), "test", &definition.FunctionConfig{
// 		Init: map[string]interface{}{
// 			"something": "To do",
// 		},
// 		Runtime: "deno",
// 	}, code)
// 	assert.NoError(t, err)

// 	err = jobInstance.Start(context.Background())
// 	assert.NoError(t, err)
// 	time.Sleep(2 * time.Second)
// 	// Some iteration logs should have been written
// 	assert.True(t, logCounter > 5)
// }
