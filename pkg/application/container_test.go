package application_test

import (
	"fmt"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/config"
)

func TestEventHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	a := assert.New(t)
	log.SetLevel(log.DebugLevel)
	cfg := config.NewConfig()
	cfg.APIBindPort = util.FindFreePort(8000)
	cfg.ClusterNatsUrl = "nats://localhost:4225"
	cfg.DataDir = os.TempDir()
	cfg.AdminToken = "1234"
	cfg.LoadApps = false

	container, err := application.NewContainer(cfg)
	a.NoError(err)
	defer container.Close()
	container.ClusterEventBus().SubscribeLogs("*.*", func(funcName, message string) {
		log.Infof("[%s] %s", funcName, message)
	})
	if err := container.Start(); err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	app, err := container.CreateApp("test")
	a.NoError(err)
	a.NoError(app.EvalString(strings.ReplaceAll(`
# events
|||yaml
"http:GET:/hello":
- MyHTTPFunc
|||


# function MyHTTPFunc
|||javascript
import {events} from "./matterless.ts";

async function handle(event) {
    return {
        status: 200,
        body: "OK"
    };
}
|||
`, "|||", "```")))

	// The actual benchmark
	for i := 0; i < 10; i++ {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test/hello", cfg.APIBindPort))
		a.NoError(err)
		a.Equal(http.StatusOK, resp.StatusCode)
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		a.NoError(err)
		a.Contains(string(data), "OK")
	}

	// t.Fail()
}
