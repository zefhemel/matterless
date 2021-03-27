package application

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
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

func NewAPIGateway(config config.Config, container *Container, functionInvokeFunc FunctionInvokeFunc) *APIGateway {
	r := mux.NewRouter()

	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%d", config.APIBindPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Debugf("Configured API APIGateway to listen on %s", srv.Addr)

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
		log.Infof("Starting API APIGateway on %s", ag.server.Addr)
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

func (ag *APIGateway) buildRouter(config config.Config) {
	ag.rootRouter.Handle("/info", expvar.Handler())
	ag.exposeRootAPI(config)
	ag.exposeStoreAPI()
	ag.exposeEventAPI()
	ag.rootRouter.HandleFunc("/{appName}/{path:.*}", func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		appName := vars["appName"]
		path := vars["path"]
		evt, err := ag.buildEvent(path, request)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(writer, "Error: %s", err)
			return
		}
		app := ag.container.Get(appName)
		if app == nil {
			http.NotFound(writer, request)
			return
		}
		response, err := app.EventBus().Call(fmt.Sprintf("http:%s:/%s", request.Method, path), evt)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(writer, "Error: %s", err)
			return
		}

		if _, ok := response.(map[string]interface{}); ok {
			// Custom handler response with plain map[string]interface{}, map back to APIGatewayResponse
			var apiGResp definition.APIGatewayResponse
			if err := mapstructure.Decode(response, &apiGResp); err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(writer, err.Error())
				return
			}
			response = &apiGResp
		}

		if apiResponse, ok := response.(*definition.APIGatewayResponse); ok {
			if apiResponse.Headers != nil {
				for name, value := range apiResponse.Headers {
					writer.Header().Set(name, value)
				}
			}
			if apiResponse.Status == 0 {
				apiResponse.Status = http.StatusOK
			}
			switch body := apiResponse.Body.(type) {
			case string:
				writer.WriteHeader(apiResponse.Status)
				writer.Write([]byte(body))
			default:
				writer.Header().Set("content-type", "application/json")
				writer.WriteHeader(apiResponse.Status)
				writer.Write([]byte(util.MustJsonString(body)))
			}
		} else {
			writer.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(writer, "Error: Did not get back an APIGatewayResponse")
			return
		}
	})
}

func (ag *APIGateway) buildEvent(path string, request *http.Request) (*definition.APIGatewayRequestEvent, error) {
	evt := &definition.APIGatewayRequestEvent{
		Path:          fmt.Sprintf("/%s", path),
		Method:        request.Method,
		Headers:       util.FlatStringMap(request.Header),
		RequestParams: util.FlatStringMap(request.URL.Query()),
	}
	if request.Header.Get("content-type") == "application/x-www-form-urlencoded" {
		if err := request.ParseForm(); err != nil {
			return nil, errors.Wrap(err, "parsing form")
		}
		evt.FormValues = util.FlatStringMap(request.PostForm)
	} else if request.Header.Get("content-type") == "application/json" {
		var jsonBody interface{}
		decoder := json.NewDecoder(request.Body)
		if err := decoder.Decode(&jsonBody); err != nil {
			return nil, errors.Wrap(err, "parsing json body")
		}
		evt.JSONBody = jsonBody
	}
	return evt, nil
}
