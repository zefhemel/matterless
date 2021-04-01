package application

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/zefhemel/matterless/pkg/store"
	"net/http"
	"strings"
)

func (ag *APIGateway) exposeStoreAPI() {
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
		if token != app.apiToken && token != ag.container.config.AdminToken {
			// Both the root token and per-app token are acceptable
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Wrong token")
			return
		}
		store.NewHTTPStore(app.dataStore).ServeHTTP(w, r)
	})
}
