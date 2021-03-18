package store_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/store"
	"github.com/zefhemel/matterless/pkg/util"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sendOp(url string, args ...interface{}) (*store.OperationResponse, error) {
	res, err := http.Post(url, "application/json", strings.NewReader(util.MustJsonString([][]interface{}{args})))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var opResponse []store.OperationResponse
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&opResponse); err != nil {
		return nil, err
	}
	return &opResponse[0], err
}

func TestHTTP(t *testing.T) {
	s, err := store.NewLevelDBStore("http_test")
	assert.NoError(t, err)
	defer s.RemoveStore()
	httpStore := store.NewHTTPStore(s)
	ts := httptest.NewServer(httpStore)
	defer ts.Close()

	response, err := sendOp(ts.URL, "put", "simpleKey", "simpleValue")
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.Status)

	response, err = sendOp(ts.URL, "get", "simpleKey")
	assert.NoError(t, err)
	assert.Equal(t, "simpleValue", response.Value)

	response, err = sendOp(ts.URL, "delete", "simpleKey")
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.Status)

	response, err = sendOp(ts.URL, "get", "simpleKey")
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, nil, response.Value)

	ps := []person{
		{
			Name: "John",
			Age:  20,
		},
		{
			Name: "Jane",
			Age:  21,
		},
	}
	for i, p := range ps {
		response, err = sendOp(ts.URL, "put", fmt.Sprintf("person:%d", i), p)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response.Status)
	}

	response, err = sendOp(ts.URL, "query-prefix", "person:")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(response.Results))

	response, err = sendOp(ts.URL, "query-range", "person:", "person:~")
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, 2, len(response.Results))
}
