package application

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/zefhemel/matterless/pkg/config"
	"io"
	"net/http"
	"strings"
)

func (ag *APIGateway) exposeRootAPI(cfg config.Config) {
	ag.rootRouter.HandleFunc("/{app}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		if !ag.rootApiAuth(w, r, cfg) {
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
			app = NewApplication(cfg, appName)
			ag.container.Register(appName, app)
		}

		if err := app.Eval(string(defBytes)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}
		fmt.Fprint(w, app.Definitions().Markdown())
	}).Methods("PUT")

	ag.rootRouter.HandleFunc("/{app}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		if !ag.rootApiAuth(w, r, cfg) {
			return
		}
		ag.container.UnRegister(appName)
		fmt.Fprint(w, "OK")
	}).Methods("DELETE")
}

func (ag *APIGateway) rootApiAuth(w http.ResponseWriter, r *http.Request, cfg config.Config) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "ROOT: No authorization provided")
		return false
	}
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || len(authHeaderParts) == 2 && authHeaderParts[0] != "bearer" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "ROOT2: No authorization provided: %+v", authHeaderParts)
		return false
	}
	token := authHeaderParts[1]
	if token != cfg.RootToken {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Wrong token")
		return false
	}
	return true
}
