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

func TestFlatStringMap(t *testing.T) {
	testMap := map[string][]string{
		"key":  {"value1", "vallue2"},
		"key2": {"value1"},
		"key3": {},
	}
	flattenedMap := util.FlatStringMap(testMap)
	assert.Equal(t, "value1", flattenedMap["key"])
	assert.Equal(t, "value1", flattenedMap["key2"])
	deepenedMap := util.ListStringMap(flattenedMap)
	assert.Equal(t, 1, len(deepenedMap["key"]))
	assert.Equal(t, "value1", deepenedMap["key"][0])
}
