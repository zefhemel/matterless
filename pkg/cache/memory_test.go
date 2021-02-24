package cache_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/cache"
)

func TestMemoryCache(t *testing.T) {
	m := cache.NewInMemoryCache()
	m.Set("name", "Test")
	assert.Equal(t, "Test", m.Get("name"))
}
