package sandbox

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type DockerInitMessage struct {
	Script string                 `json:"script"`
	Config map[string]interface{} `json:"config"`
}

// ======= Functions ============
type dockerFunctionInstance struct {
	containerName string
	controlURL    string
	serverURL     string
	lastInvoked   time.Time
	runLock       sync.Mutex
	name          string
	apiURL        string
}

var _ FunctionInstance = &dockerFunctionInstance{}

func (inst *dockerFunctionInstance) Name() string {
	return inst.name
}

func (inst *dockerFunctionInstance) LastInvoked() time.Time {
	return inst.lastInvoked
}

func newDockerFunctionInstance(ctx context.Context, cfg *config.Config, apiURL string, apiToken string, runnerType string, name string, eventBus eventbus.EventBus, functionConfig definition.FunctionConfig, code string) (*dockerFunctionInstance, error) {
	inst := &dockerFunctionInstance{
		name:          name,
		apiURL:        apiURL,
		containerName: fmt.Sprintf("mls-%s-%s", runnerType, name),
	}

	apiHost := "172.17.0.1"
	if runtime.GOOS != "linux" {
		apiHost = "host.docker.internal"
	}

	// Run "docker run -i" as child process
	cmd := exec.Command("docker", "run", "--rm", "-P", "-i",
		fmt.Sprintf("--name=%s", inst.containerName),
		fmt.Sprintf("-e API_URL=%s", fmt.Sprintf(inst.apiURL, apiHost)),
		fmt.Sprintf("-e API_TOKEN=%s", apiToken),
		"zefhemel/matterless-runner-docker", runnerType)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stdout pipe")
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stderr pipe")
	}

	// Kick off the command
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "docker run")
	}

	// Listen to the stderr and log pipes and ship everything to logChannel
	bufferedStdout := bufio.NewReader(stdoutPipe)
	bufferedStderr := bufio.NewReader(stderrPipe)
	if _, err := bufferedStderr.Peek(1); err != nil {
		log.Error("Could not peek stdout data", err)
	}

	// Send stdout and stderr to the log channel
	go inst.pipeStream(bufferedStdout, eventBus)
	go inst.pipeStream(bufferedStderr, eventBus)

	// Run "docker inspect" to fetch exposed ports
	inspectData, err := exec.Command("docker", "inspect", inst.containerName).CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "docker inspect")
	}
	var dockerInspectOutputs []struct {
		NetworkSettings struct {
			Ports map[string][]struct {
				HostPort string
			}
		}
	}
	if err := json.Unmarshal(inspectData, &dockerInspectOutputs); err != nil {
		return nil, errors.Wrap(err, "parse docker inspect")
	}

	if len(dockerInspectOutputs) == 0 {
		return nil, errors.New("invalid docker inspect output")
	}

	if len(dockerInspectOutputs[0].NetworkSettings.Ports["8081/tcp"]) == 0 {
		return nil, errors.New("invalid docker inspect output")
	}

	inst.controlURL = fmt.Sprintf("http://localhost:%s", dockerInspectOutputs[0].NetworkSettings.Ports["8081/tcp"][0].HostPort)
	inst.serverURL = fmt.Sprintf("http://localhost:%s", dockerInspectOutputs[0].NetworkSettings.Ports["8080/tcp"][0].HostPort)

	// Initialize the container via the /init call that uploads the code
	if err := inst.init(functionConfig, code); err != nil {
		return nil, err
	}

	return inst, nil
}

func (inst *dockerFunctionInstance) Kill() error {
	// Don't Kill until current run is over, if any
	inst.runLock.Lock()
	inst.runLock.Unlock()

	// Call /stop on control server, but ignore if this fails for whatever reason
	http.Get(fmt.Sprintf("%s/stop", inst.controlURL))

	// Now hard Kill the docker container, if it's still running
	exec.Command("docker", "Kill", inst.containerName).Run()

	log.Info("Killed dockerFunctionInstance.")
	return nil
}

func (inst *dockerFunctionInstance) pipeStream(bufferedReader *bufio.Reader, eventBus eventbus.EventBus) {
readLoop:
	for {
		line, err := bufferedReader.ReadString('\n')
		if err == io.EOF {
			break readLoop
		}
		if err != nil {
			log.Error("log read error", err)
			break readLoop
		}
		eventBus.Publish(fmt.Sprintf("logs:%s", inst.name), LogEntry{
			Instance: inst,
			Message:  line,
		})
	}
}

type jsError struct {
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

func (inst *dockerFunctionInstance) Invoke(ctx context.Context, event interface{}) (interface{}, error) {
	inst.runLock.Lock()
	defer inst.runLock.Unlock()

	inst.lastInvoked = time.Now()

	httpClient := http.DefaultClient
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, inst.serverURL, strings.NewReader(util.MustJsonString(event)))
	if err != nil {
		return nil, errors.Wrap(err, "invoke call")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not make HTTP invocation: %s", err.Error()))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP Error: %s", body)
	}

	var result interface{}
	jsonDecoder := json.NewDecoder(resp.Body)
	if err := jsonDecoder.Decode(&result); err != nil {
		return nil, errors.Wrap(err, "unmarshall response")
	}
	if errorMap, ok := result.(map[string]interface{}); ok {
		if errorObj, ok := errorMap["error"]; ok {
			var jsError jsError
			err = json.Unmarshal([]byte(util.MustJsonString(errorObj)), &jsError)
			if err != nil {
				return nil, fmt.Errorf("Runtime error: %s", util.MustJsonString(errorObj))
			}
			return nil, fmt.Errorf("Runtime error: %s\n%s", jsError.Message, jsError.Stack)

		}
	}

	return result, nil
}

func (inst *dockerFunctionInstance) init(functionConfig definition.FunctionConfig, code string) error {
	httpClient := http.DefaultClient
	// TODO: Remove magic 15s value
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	initMessage := DockerInitMessage{
		Config: functionConfig.Init,
		Script: code,
	}
	req, err := http.NewRequestWithContext(timeoutCtx, "POST", fmt.Sprintf("%s/init", inst.controlURL), strings.NewReader(util.MustJsonString(initMessage)))
	if err != nil {
		return errors.Wrap(err, "init call")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "init http call: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Init Error: %s", body)
	}
	return nil
}

// ======= Jobs ============

type dockerJobInstance struct {
	dockerFunctionInstance
	timeStarted time.Time
	name        string
}

var _ JobInstance = &dockerJobInstance{}

func (inst *dockerJobInstance) Name() string {
	return inst.name
}

func newDockerJobInstance(ctx context.Context, cfg *config.Config, apiURL string, apiToken string, name string, eventBus eventbus.EventBus, functionConfig definition.FunctionConfig, code string) (*dockerJobInstance, error) {
	inst := &dockerJobInstance{
		name: name,
	}

	functionInstance, err := newDockerFunctionInstance(ctx, cfg, apiURL, apiToken, "node-job", name, eventBus, functionConfig, code)
	if err != nil {
		return nil, err
	}

	inst.dockerFunctionInstance = *functionInstance

	return inst, nil
}

func (inst *dockerJobInstance) Start(ctx context.Context) error {
	inst.timeStarted = time.Now()

	httpClient := http.DefaultClient
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/start", inst.serverURL), nil)
	if err != nil {
		return errors.Wrap(err, "invoke call")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not make HTTP invocation: %s", err.Error()))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP Error: %s", body)
	}

	return nil
}

func (inst *dockerJobInstance) Stop() {
	http.Get(fmt.Sprintf("%s/stop", inst.serverURL))
}
