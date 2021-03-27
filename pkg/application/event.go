package application

import (
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/definition"
	"net/http"
	"strings"
)

func (ag *APIGateway) exposeEventAPI() {
	ag.container.eventBus.Subscribe("http:/*/_event/*", func(eventName string, eventData interface{}) (interface{}, error) {
		pieces := strings.Split(eventName, "/")
		appName := pieces[1]
		evName := pieces[3]
		httpReq, ok := eventData.(*definition.APIGatewayRequestEvent)
		if !ok {
			log.Fatal("Not an API gateway request", eventData)
		}
		authHeader := httpReq.Headers["Authorization"]
		if authHeader == "" {
			return &definition.APIGatewayResponse{
				Status: http.StatusUnauthorized,
				Body:   "No authorization provided",
			}, nil
		}
		authHeaderParts := strings.Split(authHeader, " ")
		if len(authHeaderParts) != 2 || len(authHeaderParts) == 2 && authHeaderParts[0] != "bearer" {
			return &definition.APIGatewayResponse{
				Status: http.StatusUnauthorized,
				Body:   "No authorization provided",
			}, nil
		}
		token := authHeaderParts[1]
		app := ag.container.Get(appName)
		if app == nil {
			return &definition.APIGatewayResponse{
				Status: http.StatusNotFound,
				Body:   "App not found",
			}, nil
		}
		if token != app.apiToken {
			return &definition.APIGatewayResponse{
				Status: http.StatusUnauthorized,
				Body:   "Invalid token",
			}, nil
		}
		app.EventBus().Publish(evName, httpReq.JSONBody)
		return &definition.APIGatewayResponse{
			Status: 200,
			Body: map[string]string{
				"status": "ok",
			},
		}, nil
	})
}
