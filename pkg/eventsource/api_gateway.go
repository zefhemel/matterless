package eventsource

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
	"sync"
	"time"
)

type APIGatewaySource struct {
	server             *http.Server
	rootRouter         *mux.Router
	wg                 *sync.WaitGroup
	def                *definition.APIGatewayDef
	functionInvokeFunc FunctionInvokeFunc
}

func NewAPIGatewaySource(def *definition.APIGatewayDef, functionInvokeFunc FunctionInvokeFunc) *APIGatewaySource {
	r := mux.NewRouter()

	if def.BindPort == 0 {
		def.BindPort = 8222
	}
	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%d", def.BindPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Debugf("Configured API Gateway to listen on %s", srv.Addr)

	ag := &APIGatewaySource{
		server:             srv,
		rootRouter:         r,
		wg:                 &sync.WaitGroup{},
		def:                def,
		functionInvokeFunc: functionInvokeFunc,
	}

	ag.buildRouter()

	return ag
}

func (ag *APIGatewaySource) Start() error {
	ag.wg.Add(1)
	go func() {
		defer ag.wg.Done()
		log.Infof("Starting API Gateway on port %d", ag.def.BindPort)
		if err := ag.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()
	// Wait loop to wait for server to boot
	for {
		_, err := http.Get(fmt.Sprintf("http://localhost:%d", ag.def.BindPort))
		if err == nil {
			break
		}
		log.Debug("Server still starting... ", err)
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (ag *APIGatewaySource) Stop() {
	if err := ag.server.Shutdown(context.Background()); err != nil {
		log.Fatal(err)
	}

	ag.wg.Wait()
}

type APIGatewayRequestEvent struct {
	Path          string            `json:"path"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	FormValues    map[string]string `json:"form_values"`
	RequestParams map[string]string `json:"request_params"`
	JSONBody      interface{}       `json:"json_body"`
}

type APIGatewayResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
}

func (ag *APIGatewaySource) buildHandleFunc(endPoint definition.EndpointDef) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		evt, err := ag.buildEvent(writer, request)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(writer, "Error: %s", err)
			return
		}
		result := ag.functionInvokeFunc(endPoint.Function, evt)

		var apiGatewayResp APIGatewayResponse
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
}

func (ag *APIGatewaySource) buildRouter() {
	r := ag.rootRouter
	for _, endPoint := range ag.def.Endpoints {
		route := r.HandleFunc(endPoint.Path, ag.buildHandleFunc(endPoint))

		if len(endPoint.Methods) > 0 {
			route.Methods(endPoint.Methods...)
		}
	}
}

func (ag *APIGatewaySource) buildEvent(writer http.ResponseWriter, request *http.Request) (*APIGatewayRequestEvent, error) {
	defer request.Body.Close()
	evt := &APIGatewayRequestEvent{
		Path:          request.RequestURI,
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

var _ EventSource = &APIGatewaySource{}
