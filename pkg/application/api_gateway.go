package application

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
)

type FunctionInvokeFunc func(appName string, name definition.FunctionID, event interface{}) interface{}

type APIGateway struct {
	bindPort   int
	server     *http.Server
	rootRouter *mux.Router
	wg         *sync.WaitGroup

	container *Container
	config    *config.Config
}

func init() {
	expvar.Publish("goRoutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))
}

func NewAPIGateway(config *config.Config, container *Container) *APIGateway {
	r := mux.NewRouter()

	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%d", config.APIBindPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	ag := &APIGateway{
		bindPort:   config.APIBindPort,
		container:  container,
		server:     srv,
		config:     config,
		rootRouter: r,
		wg:         &sync.WaitGroup{},
	}

	ag.buildRouter(config)

	return ag
}

func (ag *APIGateway) Start() error {
	ag.wg.Add(1)
	go func() {
		defer ag.wg.Done()
		log.Infof("Starting Matterless API Gateway on %s", ag.server.Addr)
		if err := ag.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()
	// Wait loop to wait for server to boot
	for {
		_, err := http.Get(fmt.Sprintf("http://localhost:%d", ag.bindPort))
		if err == nil {
			break
		}
		log.Debug("Server still starting... ", err)
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (ag *APIGateway) Stop() {
	if err := ag.server.Shutdown(context.Background()); err != nil {
		log.Fatal(err)
	}

	ag.wg.Wait()
}

type APIGatewayResponse struct {
	Headers map[string]string `json:"headers"`
	Status  int               `json:"status"`
	Body    interface{}       `json:"body"`
}

func (ag *APIGateway) buildRouter(config *config.Config) {
	ag.rootRouter.Handle("/info", expvar.Handler())

	// Expose internal API routes
	ag.exposeAdminAPI()
	ag.exposeStoreAPI()
	ag.exposeEventAPI()
	ag.exposeFunctionAPI()
	ag.exposeApplicationAPI()

	// Handle custom API endpoints
	ag.rootRouter.HandleFunc("/{appName}/{path:.*}", func(writer http.ResponseWriter, request *http.Request) {
		reportHTTPError := func(err error) {
			writer.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(writer, "Error: %s", err)
		}

		vars := mux.Vars(request)
		appName := vars["appName"]
		path := vars["path"]
		evt, err := ag.buildHTTPEvent(path, request)
		if err != nil {
			reportHTTPError(err)
			return
		}
		app := ag.container.Get(appName)
		if app == nil {
			http.NotFound(writer, request)
			return
		}

		log.Debugf("Received HTTP request (%s) %s", request.Method, path)

		// Perform Request via eventbus
		response, err := app.EventBus().RequestEvent(fmt.Sprintf("http:%s:/%s", request.Method, path), evt, config.HTTPGatewayResponseTimeout)
		if err != nil {
			reportHTTPError(err)
			return
		}

		// Decode response and send back
		var apiGResponse APIGatewayResponse

		if err := json.Unmarshal(response.Data, &apiGResponse); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(writer, "Error: Did not get back an properly structured object")
			return
		}

		// Headers
		for name, value := range apiGResponse.Headers {
			writer.Header().Set(name, value)
		}

		// Status code
		responseStatus := http.StatusOK
		if apiGResponse.Status != 0 {
			responseStatus = apiGResponse.Status
		}

		// Body
		switch body := apiGResponse.Body.(type) {
		case string:
			writer.WriteHeader(responseStatus)
			writer.Write([]byte(body))
		default:
			writer.Header().Set("content-type", "application/json")
			writer.WriteHeader(responseStatus)
			writer.Write([]byte(util.MustJsonString(body)))
		}
	})
}

func (ag *APIGateway) buildHTTPEvent(path string, request *http.Request) (map[string]interface{}, error) {
	evt := map[string]interface{}{
		"path":           fmt.Sprintf("/%s", path),
		"method":         request.Method,
		"headers":        util.FlatStringMap(request.Header),
		"request_params": util.FlatStringMap(request.URL.Query()),
	}
	if request.Header.Get("content-type") == "application/x-www-form-urlencoded" {
		if err := request.ParseForm(); err != nil {
			return nil, errors.Wrap(err, "parsing form")
		}
		evt["form_values"] = util.FlatStringMap(request.PostForm)
	} else if request.Header.Get("content-type") == "application/json" {
		var jsonBody interface{}
		decoder := json.NewDecoder(request.Body)
		if err := decoder.Decode(&jsonBody); err != nil {
			return nil, errors.Wrap(err, "parsing json body")
		}
		evt["json_body"] = jsonBody
	}
	return evt, nil
}

func (ag *APIGateway) authAdmin(w http.ResponseWriter, r *http.Request) bool {
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
	if token != ag.config.AdminToken {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Wrong token")
		return false
	}
	return true
}

func (ag *APIGateway) authApp(w http.ResponseWriter, r *http.Request, app *Application) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "No authorization provided")
		log.Infof("Error authenticating with event API")
		return false
	}
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || len(authHeaderParts) == 2 && authHeaderParts[0] != "bearer" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "No authorization provided")
		log.Infof("Error authenticating with event API")
		return false
	}
	token := authHeaderParts[1]

	if token != app.apiToken && token != ag.container.config.AdminToken {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Wrong token")
		return false
	}

	return true
}
