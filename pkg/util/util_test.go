package util_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/util"
	"testing"
)

func TestReverseStringSlice(t *testing.T) {
	mySlice := []string{"post1", "post3", "post2"}
	assert.Equal(t, []string{"post2", "post3", "post1"}, util.ReverseStringSlice(mySlice), "basic reversal")
	assert.Equal(t, []string{"post1", "post3", "post2"}, mySlice, "didn't change in place")
}
