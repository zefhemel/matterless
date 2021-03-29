package application

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type FunctionInvokeFunc func(appName string, name definition.FunctionID, event interface{}) interface{}

type APIGateway struct {
	bindPort   int
	server     *http.Server
	rootRouter *mux.Router
	wg         *sync.WaitGroup

	functionInvokeFunc FunctionInvokeFunc
	container          *Container
}

func init() {
	expvar.Publish("goRoutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))
}

func NewAPIGateway(config *config.Config, container *Container, functionInvokeFunc FunctionInvokeFunc) *APIGateway {
	r := mux.NewRouter()

	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%d", config.APIBindPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	ag := &APIGateway{
		bindPort:           config.APIBindPort,
		container:          container,
		server:             srv,
		rootRouter:         r,
		wg:                 &sync.WaitGroup{},
		functionInvokeFunc: functionInvokeFunc,
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

func (ag *APIGateway) buildRouter(config *config.Config) {
	ag.rootRouter.Handle("/info", expvar.Handler())

	// Expose internal API routes
	ag.exposeRootAPI(config)
	ag.exposeStoreAPI()
	ag.exposeEventAPI()

	// Handle custom API endpoints
	ag.rootRouter.HandleFunc("/{appName}/{path:.*}", func(writer http.ResponseWriter, request *http.Request) {
		reportHTTPError := func(err error) {
			writer.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(writer, "Error: %s", err)
		}

		vars := mux.Vars(request)
		appName := vars["appName"]
		path := vars["path"]
		evt, err := ag.buildEvent(path, request)
		if err != nil {
			reportHTTPError(err)
			return
		}
		app := ag.container.Get(appName)
		if app == nil {
			http.NotFound(writer, request)
			return
		}
		// TODO: Remove hardcoded timeout
		response, err := app.EventBus().Request(fmt.Sprintf("http:%s:/%s", request.Method, path), evt, 10*time.Second)
		if err != nil {
			reportHTTPError(err)
			return
		}

		if apiResponse, ok := response.(map[string]interface{}); ok {
			if headers, ok := apiResponse["headers"]; ok {
				if headerMap, ok := headers.(map[string]interface{}); ok {
					for name, value := range headerMap {
						if strVal, ok := value.(string); ok {
							writer.Header().Set(name, strVal)
						} else {
							reportHTTPError(fmt.Errorf("Header '%s' is not a string", name))
							return
						}
					}
				} else {
					reportHTTPError(errors.New("'headers' is not an object"))
					return
				}
			}
			responseStatus := http.StatusOK
			if val, ok := apiResponse["status"]; ok {
				if numberVal, ok := val.(float64); ok {
					responseStatus = int(numberVal)
				} else {
					reportHTTPError(errors.New("'status' is not a number"))
					return
				}
			}
			if bodyVal, ok := apiResponse["body"]; ok {
				switch body := bodyVal.(type) {
				case string:
					writer.WriteHeader(responseStatus)
					writer.Write([]byte(body))
				default:
					writer.Header().Set("content-type", "application/json")
					writer.WriteHeader(responseStatus)
					writer.Write([]byte(util.MustJsonString(body)))
				}
			} else {
				reportHTTPError(errors.New("No 'body' specified"))
				return
			}
		} else {
			writer.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(writer, "Error: Did not get back an object")
			return
		}
	})
}

func (ag *APIGateway) buildEvent(path string, request *http.Request) (map[string]interface{}, error) {
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
		evt["json_Body"] = jsonBody
	}
	return evt, nil
}
