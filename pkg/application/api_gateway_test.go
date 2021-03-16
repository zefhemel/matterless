package application_test

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/definition"
	"io"
	"net/http"
	"testing"
)

func TestNewHTTPServer(t *testing.T) {
	s := application.NewAPIGateway(8123,
		map[string]*application.Application{
			"test": application.NewMockApplication("test", &definition.Definitions{
				APIGateways: map[string]*definition.APIGatewayDef{
					"api_test": {
						Endpoints: []definition.EndpointDef{
							{
								Path:     "/ping",
								Methods:  []string{"GET"},
								Function: "PingFunc",
							},
							{
								Path:     "/json",
								Methods:  []string{"GET"},
								Function: "JsonFunc",
							},
						},
					},
				},
			}),
		},
		func(appName string, apigatewayName string, name definition.FunctionID, event interface{}) interface{} {
			log.Infof("Called %s with event %+v", name, event)
			if name == "PingFunc" {
				return &definition.APIGatewayResponse{
					Status: 200,
					Headers: map[string]string{
						"TestHeader": "Test",
					},
					Body: "pong",
				}
			}
			if name == "JsonFunc" {
				return &definition.APIGatewayResponse{
					Status: 200,
					Body: map[string]string{
						"name": "My name",
					},
				}
			}
			return nil
		})
	s.Start()
	resp, err := http.Get("http://127.0.0.1:8123/test/api_test/ping")
	assert.NoError(t, err)
	buf, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(buf))
	assert.Equal(t, "Test", resp.Header.Get("TestHeader"))
	resp, err = http.Get("http://127.0.0.1:8123/test/api_test/json")
	assert.NoError(t, err)
	buf, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "application/json", resp.Header.Get("content-type"))
	assert.Equal(t, `{"name":"My name"}`, string(buf))
	s.Stop()
}
