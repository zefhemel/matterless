package application

import (
	"github.com/gorilla/mux"
	"github.com/zefhemel/matterless/pkg/store"
	"net/http"
)

func (ag *APIGateway) exposeStoreAPI() {
	ag.rootRouter.HandleFunc("/{app}/_store", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		appName := vars["app"]

		app := ag.container.Get(appName)
		if app == nil {
			http.NotFound(w, r)
			return
		}

		if !ag.authApp(w, r, app) {
			return
		}

		store.NewHTTPStore(app.dataStore).ServeHTTP(w, r)
	})
}
