package application_test

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/config"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestEventHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	a := assert.New(t)
	log.SetLevel(log.DebugLevel)
	cfg := config.NewConfig()
	cfg.APIBindPort = 8123
	cfg.DataDir = os.TempDir()
	cfg.AdminToken = "1234"

	container, err := application.NewContainer(cfg)
	defer container.Close()
	container.EventBus().Subscribe("logs:*", func(eventName string, eventData interface{}) {
		log.Infof("Log: %+v", eventData)
	})
	a.NoError(err)
	app, err := application.NewApplication(cfg, "test")
	a.NoError(err)
	container.Register("test", app)
	a.NoError(app.Eval(strings.ReplaceAll(`
# Events
|||yaml
"http:GET:/hello":
- MyHTTPFunc
|||


# Function: MyHTTPFunc
|||javascript
import {respondToEvent} from "./matterless.ts";

async function handle(event) {
    await respondToEvent(event, {
        status: 200,
        body: "OK"
    });
}
|||
`, "|||", "```")))

	// The actual benchmark
	for i := 0; i < 10; i++ {
		resp, err := http.Get("http://localhost:8123/test/hello")
		a.NoError(err)
		a.Equal(http.StatusOK, resp.StatusCode)
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		a.NoError(err)
		a.Contains(string(data), "OK")
	}
}
