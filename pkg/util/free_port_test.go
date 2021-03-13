package util_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/util"
	"testing"
)

func TestFindFreePort(t *testing.T) {
	p := util.FindFreePort(8065)
	assert.NotEqual(t, -1, p)
}
