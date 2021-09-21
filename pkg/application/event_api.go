package application

import (
	"encoding/json"
	"fmt"
	"github.com/zefhemel/matterless/pkg/util"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
)

type WSEventClientMessage struct {
	Type    string `json:"type"` // authenticate | subscribe
	Token   string `json:"token,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}

type WSEventMessage struct {
	Type  string      `json:"type"` // error | event | subscribed | authenticated
	Error string      `json:"error,omitempty"`
	App   string      `json:"app,omitempty"`
	Name  string      `json:"name,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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

	// Websocket for app events
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
		// TODO: Add authentication!

		ag.eventStream(w, r, app.EventBus(), app)
	})

	ag.rootRouter.HandleFunc("/_events", func(w http.ResponseWriter, r *http.Request) {
		// Authenticate
		// TODO: Add authentication!
		ag.eventStream(w, r, ag.container.ClusterEventBus(), nil)
	})
}

func (ag *APIGateway) eventStream(w http.ResponseWriter, r *http.Request, ceb *cluster.ClusterEventBus, app *Application) {
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
	authenticated := false
messageLoop:
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			// Ignore close errors
			if _, ok := err.(*websocket.CloseError); !ok {
				log.Errorf("Websocket error: %s", err)
			}
			return
		}
		if messageType == websocket.TextMessage {
			var clientMessage WSEventClientMessage
			if err := json.Unmarshal(p, &clientMessage); err != nil {
				log.Error("Could not parse websocket message: ", err)
				continue messageLoop
			}
			switch clientMessage.Type {
			case "authenticate":
				if app != nil {
					authenticated = clientMessage.Token == app.apiToken || clientMessage.Token == ag.container.config.AdminToken
				} else {
					authenticated = clientMessage.Token == ag.container.config.AdminToken
				}
				if authenticated {
					conn.WriteMessage(websocket.TextMessage, util.MustJsonByteSlice(WSEventMessage{
						Type: "authenticated",
					}))
				} else {
					conn.WriteMessage(websocket.TextMessage, util.MustJsonByteSlice(WSEventMessage{
						Type:  "error",
						Error: "invalid-token",
					}))
				}
			case "subscribe":
				if !authenticated {
					conn.WriteMessage(websocket.TextMessage, util.MustJsonByteSlice(WSEventMessage{
						Type:  "error",
						Error: "not-authenticated",
					}))
					continue messageLoop
				}
				sub, err := ceb.SubscribeEvent(clientMessage.Pattern, func(name string, data interface{}, msg *nats.Msg) {
					subjectParts := strings.Split(msg.Subject, ".") // prefix.app.otherstufff
					conn.WriteMessage(websocket.TextMessage, util.MustJsonByteSlice(WSEventMessage{
						Type: "event",
						App:  subjectParts[1],
						Name: name,
						Data: data,
					}))
				})
				if err != nil {
					log.Error("Could not subscribe to event: ", err)
					conn.WriteMessage(websocket.TextMessage, util.MustJsonByteSlice(WSEventMessage{
						Type:  "error",
						Error: "subscription-failed",
					}))
					continue messageLoop
				}
				allSubscriptions = append(allSubscriptions, sub)
			}
		}
	}
}
