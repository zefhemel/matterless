package sandbox_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

func TestNodeSandbox(t *testing.T) {
	sillyEvent := map[string]string{
		"name": "Zef",
	}
	emptyEnv := map[string]string{}
	s := sandbox.NewNodeSandbox()
	code := `
	export default function(evt) {
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
	export default function handler() {
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

// Disabled, can only be run with local MM instance
func DisabledTestNodeSandboxClient(t *testing.T) {
	sillyEvent := map[string]string{"name": "Zef"}
	env := map[string]string{
		"URL":   "http://localhost:8065",
		"TOKEN": "MYTOKEN",
	}
	s := sandbox.NewNodeSandbox()
	code := `
	export default function(evt) {
		console.log('Starting...');
		let client = getClient();
		client.getMe().then(me => {
			console.log("Me", me);
		})
	}
	`
	_, logs, err := s.Invoke(sillyEvent, code, env)
	assert.NoError(t, err, "invoking")
	assert.Fail(t, logs)
}
