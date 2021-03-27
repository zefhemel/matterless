package application

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/zefhemel/matterless/pkg/store"
	"net/http"
	"strings"
)

func (app *Application) extendEnviron() {
	// TODO: Remove Getenv
	app.definitions.Environment["API_URL"] = fmt.Sprintf("%s/%s", app.cfg.APIExternalURL, app.appName)
	app.definitions.Environment["API_TOKEN"] = app.apiToken
}

func (ag *APIGateway) exposeAPIRoute() {
	ag.rootRouter.HandleFunc("/{app}/_store", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		appName := vars["app"]
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
		store.NewHTTPStore(app.dataStore).ServeHTTP(w, r)
	})
	//ag.rootRouter.HandleFunc("/{app}/_event/{eventName}", func(w http.ResponseWriter, r *http.Request) {
	//	defer r.Body.Close()
	//	vars := mux.Vars(r)
	//	appName := vars["app"]
	//	eventName := vars["eventName"]
	//	authHeader := r.Header.Get("Authorization")
	//	if authHeader == "" {
	//		w.WriteHeader(http.StatusUnauthorized)
	//		fmt.Fprint(w, "No authorization provided")
	//		return
	//	}
	//	authHeaderParts := strings.Split(authHeader, " ")
	//	if len(authHeaderParts) != 2 || len(authHeaderParts) == 2 && authHeaderParts[0] != "bearer" {
	//		w.WriteHeader(http.StatusUnauthorized)
	//		fmt.Fprint(w, "No authorization provided")
	//		return
	//	}
	//	token := authHeaderParts[1]
	//	app := ag.container.Get(appName)
	//	if app == nil {
	//		http.NotFound(w, r)
	//		return
	//	}
	//	if token != app.apiToken {
	//		w.WriteHeader(http.StatusUnauthorized)
	//		fmt.Fprint(w, "Wrong token")
	//		return
	//	}
	//
	//	var eventData interface{}
	//	decoder := json.NewDecoder(r.Body)
	//	if err := decoder.Decode(&eventData); err != nil {
	//		w.WriteHeader(http.StatusInternalServerError)
	//		fmt.Fprintf(w, "Error: %s", err)
	//		return
	//	}
	//	ag.container.eventBus.Publish(eventName, eventData)
	//	fmt.Fprintf(w, util.MustJsonString(struct{
	//		Status string `json:"status"`
	//	}{
	//		Status: "ok",
	//	}))
	//}).Methods("POST")
}
