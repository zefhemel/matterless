package application

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

func (ag *APIGateway) exposeApplicationAPI() {
	ag.rootRouter.HandleFunc("/{app}/_restart", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
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

		panic("TO IMPLEMENT")
		// TODO Implement again with cluster event

		fmt.Fprint(w, `{"status": "ok"}`)
	}).Methods("POST")
}
