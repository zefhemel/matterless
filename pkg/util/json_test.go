package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/util"
)

func TestJson(t *testing.T) {
	assert.Equal(t, `{"status":"ok"}`, util.MustJsonString(map[string]string{"status": "ok"}))
}
