package eventsource_test

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventsource"
	"io"
	"net/http"
	"testing"
)

func TestNewHTTPServer(t *testing.T) {
	s := eventsource.NewAPIGatewaySource(&definition.APIGatewayDef{
		BindPort: 8123,
		Endpoints: []definition.EndpointDef{
			{Path: "/ping", Methods: []string{"GET"}, Function: "Ping"},
			{Path: "/info", Methods: []string{"GET"}, Function: "Info"},
		},
	}, func(name definition.FunctionID, event interface{}) interface{} {
		log.Info("Called", name, event)
		if name == "Ping" {
			return &eventsource.APIGatewayResponse{
				Status: 200,
				Headers: map[string]string{
					"TestHeader": "Test",
				},
				Body: "pong",
			}
		}
		if name == "Info" {
			return &eventsource.APIGatewayResponse{
				Status: 200,
				Body: map[string]string{
					"name": "My name",
				},
			}
		}
		return nil
	})
	s.Start()
	resp, err := http.Get("http://127.0.0.1:8123/ping")
	assert.NoError(t, err)
	buf, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(buf))
	assert.Equal(t, "Test", resp.Header.Get("TestHeader"))
	resp, err = http.Get("http://127.0.0.1:8123/info")
	assert.NoError(t, err)
	buf, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "application/json", resp.Header.Get("content-type"))
	assert.Equal(t, `{"name":"My name"}`, string(buf))
	s.Stop()
}
