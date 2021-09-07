package application

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/util"
)

func (ag *APIGateway) exposeAdminAPI() {
	ag.rootRouter.HandleFunc("/{app}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		if !ag.authAdmin(w, r) {
			return
		}
		defBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Could not read body")
			return
		}

		app := ag.container.Get(appName)
		if app == nil {
			app, err = ag.container.CreateApp(appName)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Could not create app: %s", err)
				return
			}
		}

		if err := app.Eval(string(defBytes)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		if err := ag.container.clusterConn.Publish(cluster.EventPublishApp, []byte(util.MustJsonString(cluster.PublishApp{
			Name: appName,
			Code: string(defBytes),
		}))); err != nil {
			log.Errorf("Could not publish app registration event to cluster: %s", err)
		}

		fmt.Fprint(w, app.Definitions().Markdown())
	}).Methods("PUT")

	ag.rootRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if !ag.authAdmin(w, r) {
			return
		}

		fmt.Fprint(w, util.MustJsonString(ag.container.List()))
	}).Methods("GET")

	ag.rootRouter.HandleFunc("/{app}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		if !ag.authAdmin(w, r) {
			return
		}
		app := ag.container.Get(appName)
		if app == nil {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, app.Definitions().Markdown())
	}).Methods("GET")

	ag.rootRouter.HandleFunc("/{app}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		if !ag.authAdmin(w, r) {
			return
		}
		ag.container.Deregister(appName)

		if err := ag.container.clusterConn.Publish(cluster.EventDeleteApp, []byte(util.MustJsonString(cluster.DeleteApp{
			Name: appName,
		}))); err != nil {
			log.Errorf("Could not publish app deletion event to cluster: %s", err)
		}

		fmt.Fprint(w, "OK")
	}).Methods("DELETE")

	ag.rootRouter.HandleFunc("/{app}/_defs", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		if !ag.authAdmin(w, r) {
			return
		}
		app := ag.container.Get(appName)
		if app == nil {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, util.MustJsonString(app.Definitions()))
	}).Methods("GET")

}
