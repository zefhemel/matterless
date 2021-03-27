package application_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"io"
	"net/http"
	"testing"
)

func TestNewHTTPServer(t *testing.T) {
	cfg := config.Config{
		APIBindPort: 8123,
	}
	c := application.NewContainer(cfg)
	defer c.Close()
	c.Start()
	c.EventBus().Subscribe("http:/test/ping", func(eventName string, eventData interface{}) (interface{}, error) {
		return &definition.APIGatewayResponse{
			Status: 200,
			Headers: map[string]string{
				"TestHeader": "Test",
			},
			Body: "pong",
		}, nil

	})
	c.EventBus().Subscribe("http:/test/json", func(eventName string, eventData interface{}) (interface{}, error) {
		return &definition.APIGatewayResponse{
			Status: 200,
			Body: map[string]string{
				"name": "My name",
			},
		}, nil
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
