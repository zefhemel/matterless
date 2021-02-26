package sandbox_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

func TestMockSandbox(t *testing.T) {
	sandbox := sandbox.NewMockSandbox(map[string]string{"status": "ok"}, []string{"Test"}, nil)
	result, logs, err := sandbox.Invoke(nil, ``, map[string]string{})
	assert.Equal(t, map[string]string{"status": "ok"}, result)
	assert.Equal(t, []string{"Test"}, logs)
	assert.NoError(t, err)
}
