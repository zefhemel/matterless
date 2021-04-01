package client

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func (client *MatterlessClient) Deploy(files []string, watch bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Could not open file %s: %s", path, err)
			continue
		}
		appName := strings.Replace(filepath.Base(path), ".md", "", 1)
		fmt.Printf("Deploying %s\n", appName)
		client.updateApp(appName, string(data))
		err = watcher.Add(path)
		if err != nil {
			panic(err)
		}
	}

	if watch {
		// File watch the definition file and reload on changes
		client.watcher(watcher)
	}
}

func (client *MatterlessClient) Delete(files []string) {
	for _, path := range files {
		appName := filepath.Base(path)
		fmt.Printf("Undeploying %s\n", appName)
		client.deleteApp(appName)
	}
}

func (client *MatterlessClient) Get(appName string) (string, error) {
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
		return "", fmt.Errorf("HTTP Error: %d", resp.Status)
	}

	bodyData, _ := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
	return string(bodyData), nil
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
		return nil, fmt.Errorf("HTTP Error: %d", resp.Status)
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
		return nil, fmt.Errorf("HTTP Error (%d) %s", resp.Status, bodyData)
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

func (client *MatterlessClient) updateApp(appName, code string) {
	fmt.Printf("Updating app %s...\n", appName)

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", client.URL, appName), strings.NewReader(code))
	if err != nil {
		fmt.Println("Updating app fail: ", err)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Updating app fail: ", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyData, _ := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Error: %s", bodyData)
	} else {

		fmt.Println("All good!")
	}
}

func (client *MatterlessClient) Restart(appName string) error {
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
		return fmt.Errorf("HTTP Error (%d) %s", resp.Status, bodyData)
	}
	return nil
}

func (client *MatterlessClient) deleteApp(appName string) {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", client.URL, appName), nil)
	if err != nil {
		fmt.Println("Deleting app fail: ", err)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	if _, err := http.DefaultClient.Do(req); err != nil {
		fmt.Println("Deleting app fail: ", err)
	}
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
					log.Fatalf("Could not open file %s: %s", path, err)
					continue eventLoop
				}
				client.updateApp(filepath.Base(path), string(data))
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}
