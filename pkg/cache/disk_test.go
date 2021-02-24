package cache_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/cache"
)

func TestDiskCache(t *testing.T) {
	cacheTemp := fmt.Sprintf("%s/cache.json", os.TempDir())
	m := cache.NewOnDiskCache(cacheTemp)
	m.Set("name", "Test")
	assert.Equal(t, "Test", m.Get("name"))
	m2 := cache.NewOnDiskCache(cacheTemp)
	assert.Equal(t, "Test", m2.Get("name"))
}
