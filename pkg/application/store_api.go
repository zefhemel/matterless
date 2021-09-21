package application

import (
	"github.com/gorilla/mux"
	"github.com/zefhemel/matterless/pkg/store"
	"github.com/zefhemel/matterless/pkg/util"
	"net/http"
)

func (ag *APIGateway) exposeStoreAPI() {
	ag.rootRouter.HandleFunc("/{app}/_store", func(w http.ResponseWriter, r *http.Request) {
		var err error
		vars := mux.Vars(r)
		appName := vars["app"]

		app := ag.container.Get(appName)
		if app == nil {
			// App doesn't exist yet, however since an app may rely on required configuration
			// we should allow data to be stored before the app has been fully created, only
			// admins can do this though
			if !ag.authAdmin(w, r) {
				return
			}
			// cool, let's now create the app
			app, err = ag.container.CreateApp(appName)
			if err != nil {
				util.HTTPWriteJSONError(w, http.StatusInternalServerError, err.Error(), nil)
				return
			}
		}

		if !ag.authApp(w, r, app) {
			return
		}

		store.NewHTTPStore(app.dataStore).ServeHTTP(w, r)
	})
}
