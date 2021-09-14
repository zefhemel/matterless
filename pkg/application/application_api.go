package application

import (
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
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

		if err := ag.container.ClusterEventBus().RestartApp(appName); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			log.Infof("Restart error: %s", err.Error())
			return
		}

		fmt.Fprint(w, `{"status": "ok"}`)
	}).Methods("POST")
}
