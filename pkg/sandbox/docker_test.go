package sandbox_test

import (
	"github.com/zefhemel/matterless/pkg/util"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

func TestDockerSandbox(t *testing.T) {
	sillyEvent := map[string]string{
		"name": "Zef",
	}
	emptyEnv := map[string]string{}
	s := sandbox.NewDockerSandbox(10*time.Second, 15*time.Second)
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
	resp, logs, err := s.Invoke(sillyEvent, code, emptyEnv)
	assert.NoError(t, err, "invoking")
	assert.Equal(t, "ok", resp.(map[string]interface{})["status"], "status check")
	assert.Equal(t, "Log message", logs, "logs")

	code = `
	function handle() {
		console.log("That's all folks!");
		return {status: "ok"};
	}	
	`

	for i := 0; i < 10; i++ {
		resp, logs, err = s.Invoke(sillyEvent, code, emptyEnv)
		assert.NoError(t, err, "invoking")
		assert.Equal(t, `{"status":"ok"}`, util.MustJsonString(resp), "empty response")
		assert.Equal(t, "That's all folks!", logs, "logs")
	}

	invalidSyntax := `
		console.
	`
	_, _, err = s.Invoke(sillyEvent, invalidSyntax, emptyEnv)
	assert.Error(t, err, "invoking")
	assert.True(t, strings.Contains(err.Error(), "Unexpected identifier"), "Parse error found")

}
