package application

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
	"net/http"
	"strings"
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
	appMap             map[string]*Application
}

func NewAPIGateway(bindPort int, appMap map[string]*Application, functionInvokeFunc FunctionInvokeFunc) *APIGateway {
	r := mux.NewRouter()

	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%d", bindPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Debugf("Configured API APIGateway to listen on %s", srv.Addr)

	ag := &APIGateway{
		bindPort:           bindPort,
		appMap:             appMap,
		server:             srv,
		rootRouter:         r,
		wg:                 &sync.WaitGroup{},
		functionInvokeFunc: functionInvokeFunc,
	}

	ag.buildRouter()

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

func (ag *APIGateway) handleEndpoint(appName string, endPoint *definition.EndpointDef, writer http.ResponseWriter, request *http.Request) {
	evt, err := ag.buildEvent(request)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error: %s", err)
		return
	}
	var result interface{}
	if endPoint.Decorate != nil {
		result = endPoint.Decorate(evt, func(name definition.FunctionID, event interface{}) interface{} {
			return ag.functionInvokeFunc(appName, name, event)
		})
	} else {
		result = ag.functionInvokeFunc(appName, endPoint.Function, evt)
	}

	var apiGatewayResp definition.APIGatewayResponse
	if err := json.Unmarshal([]byte(util.MustJsonString(result)), &apiGatewayResp); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(writer, "Error: %s", err)
		return
	}
	if apiGatewayResp.Headers != nil {
		for name, value := range apiGatewayResp.Headers {
			writer.Header().Set(name, value)
		}
	}
	if apiGatewayResp.Status == 0 {
		apiGatewayResp.Status = http.StatusOK
	}
	switch body := apiGatewayResp.Body.(type) {
	case string:
		writer.WriteHeader(apiGatewayResp.Status)
		writer.Write([]byte(body))
	default:
		writer.Header().Set("content-type", "application/json")
		writer.WriteHeader(apiGatewayResp.Status)
		writer.Write([]byte(util.MustJsonString(body)))
	}
}

func (ag *APIGateway) buildRouter() {
	ag.rootRouter.HandleFunc("/info", func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		writer.Header().Set("content-type", "text/plain")
		writer.WriteHeader(http.StatusOK)
		fmt.Fprint(writer, "Endpoints exposed:\n\n")

		for appName, app := range ag.appMap {
			for _, endpointDef := range app.Definitions().APIs {
				fmt.Fprintf(writer, "/%s/%s (methods: %s)\n", appName, endpointDef.Path, strings.Join(endpointDef.Methods, ", "))
			}
		}
	})
	ag.exposeAPIRoute()
	ag.rootRouter.HandleFunc("/{app}/{endpointPath:.*}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		appName := vars["app"]
		endpointPath := fmt.Sprintf("/%s", vars["endpointPath"])
		if app, ok := ag.appMap[appName]; ok {
			handled := false
			for _, endpointDef := range app.Definitions().APIs {
				if endpointPath == endpointDef.Path {
					ag.handleEndpoint(appName, endpointDef, w, r)
					handled = true
				}
			}
			if !handled {
				w.WriteHeader(http.StatusNotFound)
				log.Infof("No handler for %s%s", appName, endpointPath)
				fmt.Fprintf(w, "No such path: %s%s", appName, endpointPath)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			log.Infof("No such app %s", appName)
			fmt.Fprintf(w, "No such app: %s", appName)
		}
	})
}

func (ag *APIGateway) buildEvent(request *http.Request) (*definition.APIGatewayRequestEvent, error) {
	vars := mux.Vars(request)
	evt := &definition.APIGatewayRequestEvent{
		Path:          fmt.Sprintf("/%s", vars["endpointPath"]),
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
