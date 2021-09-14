package application

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
)

func (ag *APIGateway) exposeAdminAPI() {
	// Application PUT
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

		code := string(defBytes)

		defs, err := definition.Check("", code, filepath.Join(ag.config.DataDir, ".importcache"))

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		// Rather than applying this locally, we'll store it just in the store, which in turn will lead to the app
		// being loaded
		if err := ag.container.Store().Put(fmt.Sprintf("app:%s", appName), defs); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		// if err := ag.container.Store().

		fmt.Fprint(w, defs.Markdown())
	}).Methods("PUT")

	ag.rootRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if !ag.authAdmin(w, r) {
			return
		}

		fmt.Fprint(w, util.MustJsonString(ag.container.List()))
	}).Methods("GET")

	ag.rootRouter.HandleFunc("/_info", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if !ag.authAdmin(w, r) {
			return
		}
		info, err := ag.container.ClusterEventBus().FetchClusterInfo(1 * time.Second)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}
		fmt.Fprint(w, util.MustJsonString(info))
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

	// Application DELETE
	ag.rootRouter.HandleFunc("/{app}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		if !ag.authAdmin(w, r) {
			return
		}

		// Check if exists
		app := ag.container.Get(appName)
		if app == nil {
			http.NotFound(w, r)
			return
		}

		// Rather than deregistering directly, we'll delete it in the store, the unregistering will be a cascading
		// effect
		if err := ag.container.Store().Delete(fmt.Sprintf("app:%s", appName)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
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
