package client

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/application"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
)

type MatterlessClient struct {
	URL    string
	Token  string
	wsConn *websocket.Conn
	done   chan struct{}
}

func NewMatterlessClient(url string, token string) *MatterlessClient {
	return &MatterlessClient{
		URL:   url,
		Token: token,
		done:  make(chan struct{}),
	}
}

func AppNameFromPath(path string) string {
	return strings.Replace(filepath.Base(path), ".md", "", 1)
}

func (client *MatterlessClient) GetAppCode(appName string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", client.URL, appName), nil)
	if err != nil {
		return "", errors.Wrap(err, "request failed")
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP Error: %s", resp.Status)
	}

	bodyData, _ := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
	return string(bodyData), nil
}

func (client *MatterlessClient) GetDefinitions(appName string) (*definition.Definitions, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s/_defs", client.URL, appName), nil)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Error: %s", resp.Status)
	}

	var defs definition.Definitions
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&defs); err != nil {
		return nil, errors.Wrap(err, "decode definitions")
	}
	return &defs, nil
}

func (client *MatterlessClient) ListApps() ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, client.URL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Error: %s", resp.Status)
	}

	bodyData, _ := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var appNames []string
	if err := json.Unmarshal(bodyData, &appNames); err != nil {
		return nil, err
	}
	return appNames, nil
}

func (client *MatterlessClient) ClusterInfo() (*cluster.ClusterInfo, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/_info", client.URL), nil)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Error: %s", resp.Status)
	}

	bodyData, _ := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var clusterInfo cluster.ClusterInfo
	if err := json.Unmarshal(bodyData, &clusterInfo); err != nil {
		return nil, err
	}
	return &clusterInfo, nil
}

func (client *MatterlessClient) storeOp(appName string, op []interface{}) (interface{}, error) {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/_store", client.URL, appName), strings.NewReader(util.MustJsonString([][]interface{}{
		op,
	})))
	req.Header.Set("content-type", "application/json")
	if err != nil {
		return nil, fmt.Errorf("store operation fail: %s", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Store operation fail: %s", err)
	}

	defer resp.Body.Close()

	bodyData, _ := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Error (%d) %s", resp.StatusCode, bodyData)
	}
	var resultObj interface{}
	if err := json.Unmarshal(bodyData, &resultObj); err != nil {
		return nil, errors.Wrapf(err, "json decoding body: %s JSON: %s", err, bodyData)
	}
	return resultObj, nil
}

func (client *MatterlessClient) StorePut(appName string, key string, value interface{}) error {
	_, err := client.storeOp(appName, []interface{}{"put", key, value})
	return err
}

func (client *MatterlessClient) StoreDel(appName string, key string) error {
	_, err := client.storeOp(appName, []interface{}{"del", key})
	return err
}

func (client *MatterlessClient) StoreGet(appName string, key string) (interface{}, error) {
	return client.storeOp(appName, []interface{}{"get", key})
}

func (client *MatterlessClient) StoreQueryPrefix(appName string, prefix string) ([][]interface{}, error) {
	result, err := client.storeOp(appName, []interface{}{"query-prefix", prefix})
	if err != nil {
		return nil, err
	}
	if resultsSlice, ok := result.([]interface{}); ok {
		if resultsObj, ok := resultsSlice[0].(map[string]interface{}); ok {
			if resultsObj, ok := resultsObj["results"]; ok {
				if resultsList, ok := resultsObj.([]interface{}); ok {
					results := make([][]interface{}, len(resultsList))
					for i := 0; i < len(resultsList); i++ {
						results[i] = resultsList[i].([]interface{})
					}
					return results, nil
				}
			} else {
				return [][]interface{}{}, nil
			}
		}
	}
	return nil, errors.New("Invalid result")
}

type ConfigIssuesError struct {
	Message      string            `json:"error"`
	ConfigIssues map[string]string `json:"data"`
}

func (mce *ConfigIssuesError) Error() string {
	return mce.Message
}

func (client *MatterlessClient) DeployApp(appName, code string) error {
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", client.URL, appName), strings.NewReader(code))
	if err != nil {
		return errors.Wrap(err, "create request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "perform request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyData, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "read response body")
		}
		if resp.Header.Get("content-type") == "application/json" {
			// May be a missing config error
			var configIssuesError ConfigIssuesError
			if err := json.Unmarshal(bodyData, &configIssuesError); err != nil {
				return errors.Wrap(err, "unmarshal config issues")
			}
			if configIssuesError.Message == "config-errors" {
				return &configIssuesError
			}
		}
		return fmt.Errorf("app update error: %s", bodyData)
	}
	return nil
}

func (client *MatterlessClient) RestartApp(appName string) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/_restart", client.URL, appName), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	bodyData, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP Error (%d) %s", resp.StatusCode, bodyData)
	}
	return nil
}

func (client *MatterlessClient) DeleteApp(appName string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", client.URL, appName), nil)
	if err != nil {
		return errors.Wrap(err, "create request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	if resp, err := http.DefaultClient.Do(req); err != nil {
		return errors.Wrap(err, "perform request")
	} else {
		defer resp.Body.Close()
		bodyData, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP Error (%d) %s", resp.StatusCode, bodyData)
		}
	}
	return nil
}

func (client *MatterlessClient) TriggerEvent(appName string, eventName string, eventData interface{}) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/_event/%s", client.URL, appName, eventName), strings.NewReader(util.MustJsonString(eventData)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	bodyData, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP Error (%d) %s", resp.StatusCode, bodyData)
	}
	return nil
}

func (client *MatterlessClient) InvokeFunction(appName string, functionName string, eventData interface{}) (interface{}, error) {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/_function/%s", client.URL, appName, functionName), strings.NewReader(util.MustJsonString(eventData)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	bodyData, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Error (%d) %s", resp.StatusCode, bodyData)
	}

	var resultObj interface{}
	if err := json.Unmarshal(bodyData, &resultObj); err != nil {
		return nil, err
	}
	return resultObj, nil
}

// TODO: Currently there can only be one connection to one app
func (client *MatterlessClient) EventStream(appName string) (chan application.WSEventMessage, error) {
	url := fmt.Sprintf("%s/%s/_events", strings.ReplaceAll(strings.ReplaceAll(client.URL, "http://", "ws://"), "https://", "wss://"), appName)

	var err error
	client.wsConn, _, err = websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "websocket connect")
	}
	if err := client.wsConn.WriteMessage(websocket.TextMessage, util.MustJsonByteSlice(application.WSEventClientMessage{
		Type:  "authenticate",
		Token: client.Token,
	})); err != nil {
		return nil, errors.Wrap(err, "send ws auth")
	}
	wsChan := make(chan application.WSEventMessage)
	go func() {
		defer close(wsChan)
	loop:
		for {
			_, message, err := client.wsConn.ReadMessage()
			if err != nil {
				log.Errorf("Error reading from websocket: %s", err)
				return
			}
			var wsMessage application.WSEventMessage
			if err := json.Unmarshal(message, &wsMessage); err != nil {
				log.Errorf("Could not unmarshal websocket message: %s", err)
				continue
			}
			select {
			case <-client.done:
				break loop
			default:
			}
			switch wsMessage.Type {
			//case "authenticated":
			//	log.Info("Event stream successfully authenticated")
			//case "subscribed":
			//	log.Info("Event stream successfully subscribed")
			case "event":
				//log.Info("Received event")
				wsChan <- wsMessage
			case "error":
				log.Errorf("Received websocket error: %s", wsMessage.Error)
			}
		}
	}()
	return wsChan, nil
}

func (client *MatterlessClient) SubscribeEvent(pattern string) error {
	if client.wsConn == nil {
		return errors.New("not connected")
	}
	return client.wsConn.WriteMessage(websocket.TextMessage, util.MustJsonByteSlice(application.WSEventClientMessage{
		Type:    "subscribe",
		Pattern: pattern,
	}))
}

func (client *MatterlessClient) Close() {
	close(client.done)
}
