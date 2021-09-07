package application

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
)

func (ag *APIGateway) exposeFunctionAPI() {
	ag.rootRouter.HandleFunc("/{app}/_function/{functionName}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		functionName := vars["functionName"]

		app := ag.container.Get(appName)
		if app == nil {
			http.NotFound(w, r)
			log.Infof("Not found app")
			return
		}

		// Authenticate
		if !ag.authApp(w, r, app) {
			return
		}

		// Decode body
		var bodyJSON interface{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&bodyJSON); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			log.Infof("Decode error: %s", err.Error())
			return
		}

		// Invoke function
		result, err := app.InvokeFunction(functionName, bodyJSON)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		// Done
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, util.MustJsonString(result))
	}).Methods(http.MethodPost)
}
