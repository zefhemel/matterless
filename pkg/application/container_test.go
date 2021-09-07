package application_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
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
	cfg.APIBindPort = 8123
	cfg.DataDir = os.TempDir()
	cfg.AdminToken = "1234"

	container, err := application.NewContainer(cfg)
	a.NoError(err)
	defer container.Close()
	container.ClusterConnection().Subscribe(fmt.Sprintf("%s.*.function.*.log", cfg.NatsPrefix), func(m *nats.Msg) {
		log.Infof("[%s] %s", m.Subject, m.Data)
	})
	app, err := container.CreateApp("test")
	a.NoError(err)
	a.NoError(app.Eval(strings.ReplaceAll(`
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
		resp, err := http.Get("http://localhost:8123/test/hello")
		a.NoError(err)
		a.Equal(http.StatusOK, resp.StatusCode)
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		a.NoError(err)
		a.Contains(string(data), "OK")
	}
}
