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
		app, ok := ag.appMap[appName]
		if !ok {
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
}
