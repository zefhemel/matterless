package application

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

func (ag *APIGateway) exposeEventAPI() {
	ag.rootRouter.HandleFunc("/{app}/_event/{eventName}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		eventName := vars["eventName"]
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "No authorization provided")
			return
		}
		authHeaderParts := strings.Split(authHeader, " ")
		if len(authHeaderParts) != 2 || len(authHeaderParts) == 2 && authHeaderParts[0] != "bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "No authorization provided")
			return
		}
		token := authHeaderParts[1]
		app := ag.container.Get(appName)
		if app == nil {
			http.NotFound(w, r)
			return
		}
		if token != app.apiToken {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Wrong token")
			return
		}
		var bodyJSON interface{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&bodyJSON); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}
		app.EventBus().Publish(eventName, bodyJSON)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}).Methods(http.MethodPost)
}
