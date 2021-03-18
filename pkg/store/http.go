package store

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/zefhemel/matterless/pkg/util"
	"net/http"
)

type HTTPStore struct {
	store Store
}

func reportHTTPError(w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, util.MustJsonString(OperationResponse{
		Status: "error",
		Error:  err.Error(),
	}))
}

type OperationResponse struct {
	Status  string          `json:"status"`
	Error   string          `json:"error,omitempty"`
	Value   interface{}     `json:"value,omitempty"`
	Results [][]interface{} `json:"results,omitempty"`
}

var _ http.Handler = &HTTPStore{}

func (s *HTTPStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if r.Header.Get("content-type") != "application/json" {
		reportHTTPError(w, http.StatusUnprocessableEntity, errors.New("Content-type needs to be application/json"))
		return
	}
	var operations [][]interface{}
	jsonDecoder := json.NewDecoder(r.Body)
	if err := jsonDecoder.Decode(&operations); err != nil {
		reportHTTPError(w, http.StatusUnprocessableEntity, err)
		return
	}
	responses := make([]OperationResponse, 0, len(operations))
operationLoop:
	for _, operation := range operations {
		operationString, ok := operation[0].(string)
		if !ok {
			responses = append(responses, OperationResponse{
				Status: "error",
				Error:  "First argument not a string",
			})
			continue operationLoop
		}
		switch operationString {
		case "put":
			if len(operation) != 3 {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "'put' operation requires two arguments",
				})
				continue operationLoop
			}
			keyString, ok := operation[1].(string)
			if !ok {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "First argument of 'put' must be a string",
				})
				continue operationLoop
			}
			if err := s.store.Put(keyString, operation[2]); err != nil {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  err.Error(),
				})
				continue operationLoop
			}
			responses = append(responses, OperationResponse{
				Status: "ok",
			})
		case "get":
			if len(operation) != 2 {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "'get' operation requires one argument",
				})
				continue operationLoop
			}
			keyString, ok := operation[1].(string)
			if !ok {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "First argument of 'get' must be a string",
				})
				continue operationLoop
			}
			val, err := s.store.Get(keyString)
			if err != nil {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  err.Error(),
				})
				continue operationLoop
			}
			responses = append(responses, OperationResponse{
				Status: "ok",
				Value:  val,
			})
		case "del":
			if len(operation) != 2 {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "'del' operation requires one argument",
				})
				continue operationLoop
			}
			keyString, ok := operation[1].(string)
			if !ok {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "First argument of 'del' must be a string",
				})
				continue operationLoop
			}
			if err := s.store.Delete(keyString); err != nil {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  err.Error(),
				})
				continue operationLoop
			}
			responses = append(responses, OperationResponse{
				Status: "ok",
			})
		case "query-prefix":
			if len(operation) != 2 {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "'query-prefix' operation requires one argument",
				})
				continue operationLoop
			}
			queryPrefix, ok := operation[1].(string)
			if !ok {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "First argument of 'query-prefix' must be a string",
				})
				continue operationLoop
			}
			val, err := s.store.QueryPrefix(queryPrefix)
			if err != nil {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  err.Error(),
				})
				continue operationLoop
			}
			results := queryResultsToHTTPSlice(val)
			responses = append(responses, OperationResponse{
				Status:  "ok",
				Results: results,
			})
		case "query-range":
			if len(operation) != 3 {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "'query-range' operation requires three arguments",
				})
				continue operationLoop
			}
			fromRange, ok := operation[1].(string)
			if !ok {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "First argument of 'query-range' must be a string",
				})
				continue operationLoop
			}
			toRange, ok := operation[2].(string)
			if !ok {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  "Second argument of 'query-range' must be a string",
				})
				continue operationLoop
			}
			val, err := s.store.QueryRange(fromRange, toRange)
			if err != nil {
				responses = append(responses, OperationResponse{
					Status: "error",
					Error:  err.Error(),
				})
				continue operationLoop
			}
			results := queryResultsToHTTPSlice(val)
			responses = append(responses, OperationResponse{
				Status:  "ok",
				Results: results,
			})
		default:
			responses = append(responses, OperationResponse{
				Status: "error",
				Error:  fmt.Sprintf("Invalid operation: %s", operationString),
			})
		}
	}
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(responses); err != nil {
		fmt.Fprintf(w, "Could not JSON encode: %s", err)
	}
}

func queryResultsToHTTPSlice(val []QueryResult) [][]interface{} {
	results := make([][]interface{}, len(val))
	for i := 0; i < len(val); i++ {
		results[i] = []interface{}{val[i].Key, val[i].Value}
	}
	return results
}

func NewHTTPStore(store Store) *HTTPStore {
	return &HTTPStore{
		store: store,
	}
}
