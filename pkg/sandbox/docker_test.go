package sandbox_test

import (
	"context"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

func TestDockerSandbox(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	sillyEvent := map[string]string{
		"name": "Zef",
	}
	s := sandbox.NewDockerSandbox(10*time.Second, 15*time.Second)
	go func() {
		for logEntry := range s.Logs() {
			log.Infof("Got log: %s", logEntry.Message)
		}
	}()
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
