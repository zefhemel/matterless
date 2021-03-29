package application_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/config"
	"io"
	"net/http"
	"os"
	"testing"
)

func TestNewHTTPServer(t *testing.T) {
	cfg := config.Config{
		APIBindPort: 8123,
		DataDir:     os.TempDir(),
	}
	c, err := application.NewContainer(cfg)
	assert.NoError(t, err)
	defer c.Close()
	app := application.NewMockApplication("test")
	c.Register("test", app)
	app.EventBus().Subscribe("http:GET:/ping", func(eventName string, eventData interface{}) {
		app.EventBus().Respond(eventData, map[string]interface{}{
			"status": float64(200),
			"headers": map[string]interface{}{
				"TestHeader": "Test",
			},
			"body": "pong",
		})
	})
	app.EventBus().Subscribe("http:GET:/json", func(eventName string, eventData interface{}) {
		app.EventBus().Respond(eventData, map[string]interface{}{
			"status": float64(200),
			"body": map[string]string{
				"name": "My name",
			},
		})
	})

	resp, err := http.Get("http://127.0.0.1:8123/test/ping")
	assert.NoError(t, err)
	buf, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(buf))
	assert.Equal(t, "Test", resp.Header.Get("TestHeader"))
	resp, err = http.Get("http://127.0.0.1:8123/test/json")
	assert.NoError(t, err)
	buf, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("content-type"))
	assert.Equal(t, `{"name":"My name"}`, string(buf))
}
