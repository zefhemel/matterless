package application

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

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
		app.EventBus().Publish(eventName, bodyJSON)

		// Done
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}).Methods(http.MethodPost)
}
