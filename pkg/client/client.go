package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
)

type MatterlessClient struct {
	URL   string
	Token string
}

func NewMatterlessClient(url string, token string) *MatterlessClient {
	return &MatterlessClient{
		URL:   url,
		Token: token,
	}
}

func AppNameFromPath(path string) string {
	return strings.Replace(filepath.Base(path), ".md", "", 1)
}

func (client *MatterlessClient) DeployAppFiles(files []string, watch bool) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "read file: %s", path)
		}
		defs, err := definition.Check(path, string(data), "")
		if err != nil {
			log.Fatal(err)
		}

		//log.Infof("Here are all defs: %+v", defs.Events)

		if err := client.DeployApp(AppNameFromPath(path), defs.Markdown()); err != nil {
			return errors.Wrap(err, "deploy")
		}
		err = watcher.Add(path)
		if err != nil {
			return errors.Wrap(err, "watcher")
		}
	}

	if watch {
		// File watch the definition file and reload on changes
		go client.watcher(watcher)
	}

	return nil
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

func (client *MatterlessClient) watcher(watcher *fsnotify.Watcher) {
eventLoop:
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				path := event.Name
				log.Infof("Definition %s modified, reloading...", path)
				data, err := os.ReadFile(path)
				if err != nil {
					log.Error(err)
					continue eventLoop
				}
				defs, err := definition.Check(path, string(data), "")
				if err != nil {
					log.Error(err)
					continue eventLoop
				}

				if err := client.DeployApp(AppNameFromPath(path), defs.Markdown()); err != nil {
					log.Errorf("Could not redeploy app %s: %s", AppNameFromPath(path), err)
					continue eventLoop
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Errorf("Watcher error: %s", err)
		}
	}
}
