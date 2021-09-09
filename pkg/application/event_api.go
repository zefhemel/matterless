package application

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
)

type subscribeMessage struct {
	Pattern string `json:"pattern"`
}

type eventMessage struct {
	EventName string      `json:"name"`
	Data      interface{} `json:"data"`
}

func (ag *APIGateway) exposeEventAPI() {
	ag.rootRouter.HandleFunc("/{app}/_event/{eventName}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		eventName := vars["eventName"]

		app := ag.container.Get(appName)

		if app == nil {
			http.NotFound(w, r)
			log.Debugf("Not found app: %s", appName)
			return
		}

		// Authenticate
		if !ag.authApp(w, r, app) {
			return
		}

		// Parse event from body
		var bodyJSON interface{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&bodyJSON); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			log.Debugf("Decode error: %s", err.Error())
			return
		}

		// Publish event
		if err := app.PublishAppEvent(eventName, bodyJSON); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			log.Debugf("Could not publish: %s", err.Error())
			return
		}

		// Done
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}).Methods(http.MethodPost)

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// Websocket event listening
	ag.rootRouter.HandleFunc("/{app}/_events", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		appName := vars["app"]
		app := ag.container.Get(appName)

		if app == nil {
			http.NotFound(w, r)
			log.Debugf("Not found app: %s", appName)
			return
		}

		// Authenticate
		//if !ag.authApp(w, r, app) {
		//	return
		//}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("websocket error: %s", err), http.StatusBadRequest)
			return
		}
		defer conn.Close()
		allSubscriptions := []cluster.Subscription{}
		defer func() {
			// Clean up all subscriptions
			for _, subscription := range allSubscriptions {
				subscription.Unsubscribe()
			}
		}()
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Errorf("Websocket error: %s", err)
				return
			}
			if messageType == websocket.TextMessage {
				var subscribeMessage subscribeMessage
				if err := json.Unmarshal(p, &subscribeMessage); err != nil {
					log.Error("Could not parse websocket message: ", err)
					continue
				}
				sub, err := app.EventBus().Subscribe(subscribeMessage.Pattern, func(msg *nats.Msg) {
					conn.WriteMessage(websocket.TextMessage, msg.Data)
				})
				if err != nil {
					log.Error("Could not subscribe to event: ", err)
					continue
				}
				allSubscriptions = append(allSubscriptions, sub)
			}
		}
	})
}
