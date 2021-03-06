package sandbox_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

func DisabledTestNodeDockerSandbox(t *testing.T) {
	sillyEvent := map[string]string{
		"name": "Zef",
	}
	emptyEnv := map[string]string{}
	s := sandbox.NewNodeDockerSandbox()
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
	}	
	`
	resp, logs, err = s.Invoke(sillyEvent, code, emptyEnv)
	assert.NoError(t, err, "invoking")
	assert.Equal(t, map[string]interface{}{}, resp, "empty response")
	assert.Equal(t, "That's all folks!", logs, "logs")

	invalidSyntax := `
		console.
	`
	_, _, err = s.Invoke(sillyEvent, invalidSyntax, emptyEnv)
	assert.Error(t, err, "invoking")
}
